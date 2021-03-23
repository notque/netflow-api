/*******************************************************************************
*
* Copyright 2017 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

package api

import (
	"net/http"

	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/notque/netflow-api/pkg/netflow"
	"github.com/notque/netflow-api/pkg/util"
	"github.com/pkg/errors"
)

// EventList is the model for JSON returned by the ListEvents API call
type EventList struct {
	NextURL string               `json:"next,omitempty"`
	PrevURL string               `json:"previous,omitempty"`
	Events  []*netflow.ListEvent `json:"events"`
	Total   int                  `json:"total"`
}

//ListEvents handles GET /v1/events.
func (p *v1Provider) ListEvents(res http.ResponseWriter, req *http.Request) {
	util.LogDebug("* api.ListEvents: Check token")
	token := p.CheckToken(req)
	if !token.Require(res, "event:list") {
		return
	}

	// QueryParams
	// Parse the integers for offset & limit
	offset, _ := strconv.ParseUint(req.FormValue("offset"), 10, 32)
	limit, _ := strconv.ParseUint(req.FormValue("limit"), 10, 32)

	// Parse the sort query string
	//slice of a struct, key and direction.

	sortSpec := []netflow.FieldOrder{}
	validSortTopics := map[string]bool{"time": true, "initiator_id": true, "observer_type": true, "target_type": true,
		"target_id": true, "action": true, "outcome": true, "initiator_name": true, "initiator_type": true,

		// deprecated
		"source": true, "resource_type": true, "resource_name": true, "event_type": true}
	validSortDirection := map[string]bool{"asc": true, "desc": true}
	sortParam := req.FormValue("sort")

	if sortParam != "" {
		for _, sortElement := range strings.Split(sortParam, ",") {
			keyVal := strings.SplitN(sortElement, ":", 2)
			//`time`, `source`, `resource_type`, `resource_name`, and `event_type`.
			sortfield := keyVal[0]
			if !validSortTopics[sortfield] {
				err := fmt.Errorf("not a valid topic: %s, valid topics: %v", sortfield, reflect.ValueOf(validSortTopics).MapKeys())
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			defsortorder := "asc"
			if len(keyVal) == 2 {
				sortDirection := keyVal[1]
				if !validSortDirection[sortDirection] {
					err := fmt.Errorf("sort direction %s is invalid, must be asc or desc", sortDirection)
					http.Error(res, err.Error(), http.StatusBadRequest)
					return
				}
				defsortorder = sortDirection
			}

			s := netflow.FieldOrder{Fieldname: sortfield, Order: defsortorder}
			sortSpec = append(sortSpec, s)

		}
	}

	// Next, parse the elements of the time range filter
	timeRange := make(map[string]string)
	validOperators := map[string]bool{"lt": true, "lte": true, "gt": true, "gte": true}
	timeParam := req.FormValue("time")
	if timeParam != "" {
		for _, timeElement := range strings.Split(timeParam, ",") {
			keyVal := strings.SplitN(timeElement, ":", 2)
			operator := keyVal[0]
			if !validOperators[operator] {
				err := fmt.Errorf("time operator %s is not valid. Must be lt, lte, gt or gte", operator)
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			_, exists := timeRange[operator]
			if exists {
				err := fmt.Errorf("time operator %s can only occur once", operator)
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			if len(keyVal) != 2 {
				err := fmt.Errorf("time operator %s missing :<timestamp>", operator)
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			validTimeFormats := []string{time.RFC3339, "2006-01-02T15:04:05-0700", "2006-01-02T15:04:05"}
			var isValidTimeFormat bool
			timeStr := keyVal[1]
			for _, timeFormat := range validTimeFormats {
				_, err := time.Parse(timeFormat, timeStr)
				if err != nil {
					isValidTimeFormat = true
					break
				}
			}
			if !isValidTimeFormat {
				err := fmt.Errorf("invalid time format: %s", timeStr)
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			timeRange[operator] = timeStr
		}
	}

	util.LogDebug("api.ListEvents: Create filter")
	filter := netflow.EventFilter{
		ObserverType:  req.FormValue("observer_type") + req.FormValue("source"),
		TargetType:    req.FormValue("target_type") + req.FormValue("resource_type"),
		TargetID:      req.FormValue("target_id"),
		InitiatorID:   req.FormValue("initiator_id") + req.FormValue("user_name"),
		InitiatorType: req.FormValue("initiator_type"),
		InitiatorName: req.FormValue("initiator_name"),
		Action:        req.FormValue("action") + req.FormValue("event_type"),
		Outcome:       req.FormValue("outcome"),
		Time:          timeRange,
		Offset:        uint(offset),
		Limit:         uint(limit),
		Sort:          sortSpec,
	}

	util.LogDebug("api.ListEvents: call netflow-api.GetEvents()")
	indexID, err := getIndexID(token, req, res)
	if err != nil {
		return
	}
	events, total, err := netflow.GetEvents(&filter, indexID, p.keystone, p.storage)
	if ReturnError(res, err) {
		util.LogError("api.ListEvents: error %s", err)
		storageErrorsCounter.Add(1)
		return
	}

	eventList := EventList{Events: events, Total: total}

	// What protocol to use for PrevURL and NextURL?
	protocol := getProtocol(req)
	// Do we need a NextURL?
	if int(filter.Offset+filter.Limit) < total {
		req.Form.Set("offset", strconv.FormatUint(uint64(filter.Offset+filter.Limit), 10))
		eventList.NextURL = fmt.Sprintf("%s://%s%s?%s", protocol, req.Host, req.URL.Path, req.Form.Encode())
	}
	// Do we need a PrevURL?
	if int(filter.Offset-filter.Limit) >= 0 {
		req.Form.Set("offset", strconv.FormatUint(uint64(filter.Offset-filter.Limit), 10))
		eventList.PrevURL = fmt.Sprintf("%s://%s%s?%s", protocol, req.Host, req.URL.Path, req.Form.Encode())
	}

	ReturnJSON(res, http.StatusOK, eventList)
}

func getProtocol(req *http.Request) string {
	protocol := "http"
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		protocol = "https"
	}
	return protocol
}

//GetEvent handles GET /v1/events/:event_id.
func (p *v1Provider) GetEventDetails(res http.ResponseWriter, req *http.Request) {
	token := p.CheckToken(req)
	if !token.Require(res, "event:show") {
		return
	}
	eventID := mux.Vars(req)["event_id"]
	indexID, err := getIndexID(token, req, res)
	if err != nil {
		return
	}

	event, err := netflow.GetEvent(eventID, indexID, p.keystone, p.storage)

	if ReturnError(res, err) {
		util.LogError("error getting events from Storage: %s", err)
		storageErrorsCounter.Add(1)
		return
	}
	if event == nil {
		err := fmt.Errorf("event %s could not be found in project %s", eventID, indexID)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}
	ReturnJSON(res, http.StatusOK, event)
}

//GetAttributes handles GET /v1/attributes/:attribute_name
func (p *v1Provider) GetAttributes(res http.ResponseWriter, req *http.Request) {
	token := p.CheckToken(req)
	if !token.Require(res, "event:show") {
		return
	}

	// Handle QueryParams
	queryName := mux.Vars(req)["attribute_name"]
	if queryName == "" {
		util.LogDebug("attribute_name empty")
		return
	}
	maxdepth, _ := strconv.ParseUint(req.FormValue("max_depth"), 10, 32)
	limit, _ := strconv.ParseUint(req.FormValue("limit"), 10, 32)
	// Default Limit of 50 if not specified by queryparam
	if limit == 0 {
		limit = 50
	}

	util.LogDebug("api.GetAttributes: Create filter")
	filter := netflow.AttributeFilter{
		QueryName: queryName,
		MaxDepth:  uint(maxdepth),
		Limit:     uint(limit),
	}

	indexID, err := getIndexID(token, req, res)
	if err != nil {
		return
	}

	attribute, err := netflow.GetAttributes(&filter, indexID, p.storage)

	if ReturnError(res, err) {
		util.LogError("could not get attributes from Storage: %s", err)
		storageErrorsCounter.Add(1)
		return
	}
	if attribute == nil {
		err := fmt.Errorf("attribute %s could not be found in project %s", attribute, indexID)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}
	ReturnJSON(res, http.StatusOK, attribute)
}

func getIndexID(token *Token, r *http.Request, w http.ResponseWriter) (string, error) {
	// Get index ID from a token
	// Defaults to a token project scope
	indexID := token.context.Auth["project_id"]
	if indexID == "" {
		// Fallback to a token domain scope
		indexID = token.context.Auth["domain_id"]
	}

	// Whem the project_id argument is defined, check for the cluster_viewer rule
	if v := r.FormValue("project_id"); v != "" {
		if !token.Require(w, "cluster_viewer") {
			// not a cloud admin, no possibility to override indexID
			return "", errors.New("cannot override index ID")
		}
		// Index ID can be overridden with a query parameter, when a cluster_viewer rule is used
		return v, nil
	}

	return indexID, nil
}
