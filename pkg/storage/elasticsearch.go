package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sapcc/hermes/pkg/util"
	"github.com/spf13/viper"
	elastic "gopkg.in/olivere/elastic.v5"
)

type elasticSearch struct {
	client *elastic.Client
}

var es elasticSearch

// Initialise and return the ES driver
func ElasticSearch() Driver {
	if es.client == nil {
		es.init()
	}
	return es
}

func (es *elasticSearch) init() {
	util.LogDebug("Initiliasing ElasticSearch()")

	// Create a client
	var err error
	var url = viper.GetString("elasticsearch.url")
	util.LogDebug("Using ElasticSearch URL: %s", url)
	es.client, err = elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		panic(err)
	}
}

func (es elasticSearch) GetEvents(filter *Filter, tenantId string) ([]*EventDetail, int, error) {
	index := indexName(tenantId)
	util.LogDebug("Looking for events in index %s", index)

	query := elastic.NewBoolQuery()
	if filter.Source != "" {
		query = query.Filter(elastic.NewMatchPhrasePrefixQuery("publisher_id", filter.Source))
	}
	if filter.ResourceType != "" {
		query = query.Filter(elastic.NewMatchPhrasePrefixQuery("payload.target.typeURI", filter.ResourceType))
	}
	if filter.ResourceId != "" {
		query = query.Filter(elastic.NewTermQuery("payload.target.id.raw", filter.ResourceId))
	}
	if filter.UserId != "" {
		query = query.Filter(elastic.NewTermQuery("payload.initiator.user_id.raw", filter.UserId))
	}
	if filter.EventType != "" {
		query = query.Filter(elastic.NewMatchPhrasePrefixQuery("event_type", filter.EventType))
	}
	if filter.Time != nil && len(filter.Time) > 0 {
		for key, value := range filter.Time {
			timeField := "payload.eventTime"
			switch key {
			case "lt":
				query = query.Filter(elastic.NewRangeQuery(timeField).Lt(value))
			case "lte":
				query = query.Filter(elastic.NewRangeQuery(timeField).Lte(value))
			case "gt":
				query = query.Filter(elastic.NewRangeQuery(timeField).Gt(value))
			case "gte":
				query = query.Filter(elastic.NewRangeQuery(timeField).Gte(value))
			}
		}
	}
	//Mapping from RequestSort parameter
	esFieldMapping := map[string]string{
		"time":          "payload.eventTime",
		"source":        "publisher_id",
		"resource_type": "payload.target.typeURI",
		"resource_name": "payload.target.id.raw",
		"event_type":    "event_type",
	}

	esSearch := es.client.Search().
		Index(index).
		Query(query)

	if filter.Sort != nil {
		for _, fieldOrder := range filter.Sort {
			switch fieldOrder.Order {
			case "asc":
				esSearch = esSearch.Sort(esFieldMapping[fieldOrder.Fieldname], true)
			case "desc":
				esSearch = esSearch.Sort(esFieldMapping[fieldOrder.Fieldname], false)
			}
		}
	}

	esSearch = esSearch.
		Sort("@timestamp", false).
		From(int(filter.Offset)).Size(int(filter.Limit))

	searchResult, err := esSearch.Do(context.Background()) // execute
	if err != nil {
		return nil, 0, err
	}

	util.LogDebug("Got %d hits", searchResult.TotalHits())

	//Construct EventDetail array from search results
	var events []*EventDetail
	for _, hit := range searchResult.Hits.Hits {
		var de EventDetail
		err := json.Unmarshal(*hit.Source, &de)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, &de)
	}
	total := searchResult.TotalHits()

	return events, int(total), nil
}

func (es elasticSearch) GetEvent(eventId string, tenantId string) (*EventDetail, error) {
	index := indexName(tenantId)
	util.LogDebug("Looking for event %s in index %s", eventId, index)

	query := elastic.NewTermQuery("payload.id.raw", eventId)
	esSearch := es.client.Search().
		Index(index).
		Query(query)

	searchResult, err := esSearch.Do(context.Background())
	if err != nil {
		return nil, err
	}
	total := searchResult.TotalHits()

	if total > 0 {
		hit := searchResult.Hits.Hits[0]
		var de EventDetail
		err := json.Unmarshal(*hit.Source, &de)
		return &de, err
	}
	return nil, nil
}

func (es elasticSearch) MaxLimit() uint {
	return uint(viper.GetInt("elasticsearch.max_result_window"))
}

func indexName(tenantId string) string {
	index := "audit-*"
	if tenantId != "" {
		index = fmt.Sprintf("audit-%s-*", tenantId)
	}
	return index
}
