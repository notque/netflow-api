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

package netflow

import (
	"fmt"

	"github.com/jinzhu/copier"
	"github.com/notque/netflow-api/pkg/cadf"
	"github.com/notque/netflow-api/pkg/identity"
	"github.com/notque/netflow-api/pkg/storage"
	"github.com/notque/netflow-api/pkg/util"
)

// ListEvent contains high-level data about an event, intended as a list item
//  The JSON annotations here are for the JSON to be returned by the API
type ListEvent struct {
	ID        string      `json:"id"`
	Time      string      `json:"eventTime"`
	Action    string      `json:"action"`
	Outcome   string      `json:"outcome"`
	Initiator ResourceRef `json:"initiator"`
	Target    ResourceRef `json:"target"`
	Observer  ResourceRef `json:"observer"`
}

//ResourceRef is an embedded struct for ListEvents (eg. Initiator, Target, Observer)
type ResourceRef struct {
	TypeURI string `json:"typeURI"`
	ID      string `json:"id"`
	Name    string `json:"name"`
}

// EventFilter maps to the filtering/paging/sorting allowed by the API for Events
type EventFilter struct {
	ObserverType  string
	TargetType    string
	TargetID      string
	InitiatorID   string
	InitiatorType string
	InitiatorName string
	Action        string
	Outcome       string
	Time          map[string]string
	Offset        uint
	Limit         uint
	Sort          []FieldOrder
}

// FieldOrder is an embedded struct for Event Filtering
type FieldOrder struct {
	Fieldname string
	Order     string //asc or desc
}

// AttributeFilter maps to the filtering allowed by the API for Attributes
type AttributeFilter struct {
	QueryName string
	MaxDepth  uint
	Limit     uint
}

// GetEvents returns a list of matching events (with filtering)
func GetEvents(filter *EventFilter, tenantID string, keystoneDriver identity.Identity, eventStore storage.Storage) ([]*ListEvent, int, error) {
	storageFilter, err := storageFilter(filter, keystoneDriver, eventStore)
	if err != nil {
		return nil, 0, err
	}
	util.LogDebug("netflow-api.GetEvents: tenant id is %s", tenantID)
	eventDetails, total, err := eventStore.GetEvents(storageFilter, tenantID)
	if err != nil {
		return nil, 0, err
	}
	events, err := eventsList(eventDetails, keystoneDriver)
	if err != nil {
		return nil, 0, err
	}
	return events, total, err
}

func storageFilter(filter *EventFilter, keystoneDriver identity.Identity, eventStore storage.Storage) (*storage.EventFilter, error) {
	// As per the documentation, the default limit is 10
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	if filter.Offset+filter.Limit > eventStore.MaxLimit() {
		return nil, fmt.Errorf("offset %d plus limit %d exceeds the maximum of %d",
			filter.Offset, filter.Limit, eventStore.MaxLimit())
	}

	var storageFieldOrder []storage.FieldOrder
	err := copier.Copy(&storageFieldOrder, &filter.Sort)
	if err != nil {
		panic("Could not copy storage field order.")
	}
	storageFilter := storage.EventFilter{
		ObserverType:  filter.ObserverType,
		InitiatorID:   filter.InitiatorID,
		InitiatorType: filter.InitiatorType,
		InitiatorName: filter.InitiatorName,
		TargetType:    filter.TargetType,
		TargetID:      filter.TargetID,
		Action:        filter.Action,
		Outcome:       filter.Outcome,
		Time:          filter.Time,
		Offset:        filter.Offset,
		Limit:         filter.Limit,
		Sort:          storageFieldOrder,
	}
	return &storageFilter, nil
}

// eventsList Construct ListEvents
func eventsList(eventDetails []*cadf.Event, keystoneDriver identity.Identity) ([]*ListEvent, error) {
	var events []*ListEvent
	for _, storageEvent := range eventDetails {
		event := ListEvent{
			Initiator: ResourceRef{
				TypeURI: storageEvent.Initiator.TypeURI,
				ID:      storageEvent.Initiator.ID,
				Name:    storageEvent.Initiator.Name,
			},
			Target: ResourceRef{
				TypeURI: storageEvent.Target.TypeURI,
				ID:      storageEvent.Target.ID,
			},
			ID:      storageEvent.ID,
			Action:  storageEvent.Action,
			Outcome: storageEvent.Outcome,
			Time:    storageEvent.EventTime,
			Observer: ResourceRef{
				TypeURI: storageEvent.Observer.TypeURI,
				ID:      storageEvent.Observer.ID,
				Name:    storageEvent.Initiator.Name,
			},
		}
		err := copier.Copy(&event.Initiator, &storageEvent.Initiator)
		if err != nil {
			return nil, err
		}

		events = append(events, &event)
	}
	return events, nil
}

// GetEvent returns the CADF detail for event with the specified ID
func GetEvent(eventID string, tenantID string, keystoneDriver identity.Identity, eventStore storage.Storage) (*cadf.Event, error) {
	event, err := eventStore.GetEvent(eventID, tenantID)

	return event, err
}

//GetAttributes No Logic here, but handles mock implementation for eventStore
func GetAttributes(filter *AttributeFilter, tenantID string, eventStore storage.Storage) ([]string, error) {
	attributeFilter := storage.AttributeFilter{
		QueryName: filter.QueryName,
		MaxDepth:  filter.MaxDepth,
		Limit:     filter.Limit,
	}
	attribute, err := eventStore.GetAttributes(&attributeFilter, tenantID)

	return attribute, err
}
