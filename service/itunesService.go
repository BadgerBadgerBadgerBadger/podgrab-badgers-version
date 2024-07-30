package service

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/TheHippo/podcastindex"
	"github.com/akhilrex/podgrab/model"
	pkgErrors "github.com/pkg/errors"
)

type SearchService interface {
	Query(q string) ([]*model.CommonSearchResultModel, error)
}

type ItunesService struct {
}

const ItunesBase = "https://itunes.apple.com"

func (service ItunesService) Query(q string) ([]*model.CommonSearchResultModel, error) {
	u := fmt.Sprintf("%s/search?term=%s&entity=podcast", ItunesBase, url.QueryEscape(q))

	body, _ := makeQuery(u)
	var response model.ItunesResponse

	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "failed to unmarshal itunes response")
	}

	var toReturn []*model.CommonSearchResultModel

	for _, obj := range response.Results {
		toReturn = append(toReturn, GetSearchFromItunes(obj))
	}

	return toReturn, nil
}

type PodcastIndexService struct {
}

const (
	PodcastIndexKey    = "LNGTNUAFVL9W2AQKVZ49"
	PodcastIndexSecret = "H8tq^CZWYmAywbnngTwB$rwQHwMSR8#fJb#Bhgb3"
)

func (service PodcastIndexService) Query(q string) ([]*model.CommonSearchResultModel, error) {

	c := podcastindex.NewClient(PodcastIndexKey, PodcastIndexSecret)
	var toReturn []*model.CommonSearchResultModel

	podcasts, err := c.Search(q)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "failed to search podcast index")
	}

	for _, obj := range podcasts {
		toReturn = append(toReturn, GetSearchFromPodcastIndex(obj))
	}

	return toReturn, nil
}
