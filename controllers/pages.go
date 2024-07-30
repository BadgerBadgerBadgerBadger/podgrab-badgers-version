package controllers

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/akhilrex/podgrab/db"
	"github.com/akhilrex/podgrab/model"
	"github.com/akhilrex/podgrab/service"
	"github.com/gin-gonic/gin"
	pkgErrors "github.com/pkg/errors"
)

type SearchGPodderData struct {
	Q            string `binding:"required" form:"q" json:"q" query:"q"`
	SearchSource string `binding:"required" form:"searchSource" json:"searchSource" query:"searchSource"`
}
type SettingModel struct {
	DownloadOnAdd                 bool   `form:"downloadOnAdd" json:"downloadOnAdd" query:"downloadOnAdd"`
	InitialDownloadCount          int    `form:"initialDownloadCount" json:"initialDownloadCount" query:"initialDownloadCount"`
	AutoDownload                  bool   `form:"autoDownload" json:"autoDownload" query:"autoDownload"`
	AppendDateToFileName          bool   `form:"appendDateToFileName" json:"appendDateToFileName" query:"appendDateToFileName"`
	AppendEpisodeNumberToFileName bool   `form:"appendEpisodeNumberToFileName" json:"appendEpisodeNumberToFileName" query:"appendEpisodeNumberToFileName"`
	DarkMode                      bool   `form:"darkMode" json:"darkMode" query:"darkMode"`
	DownloadEpisodeImages         bool   `form:"downloadEpisodeImages" json:"downloadEpisodeImages" query:"downloadEpisodeImages"`
	GenerateNFOFile               bool   `form:"generateNFOFile" json:"generateNFOFile" query:"generateNFOFile"`
	DontDownloadDeletedFromDisk   bool   `form:"dontDownloadDeletedFromDisk" json:"dontDownloadDeletedFromDisk" query:"dontDownloadDeletedFromDisk"`
	BaseUrl                       string `form:"baseUrl" json:"baseUrl" query:"baseUrl"`
	MaxDownloadConcurrency        int    `form:"maxDownloadConcurrency" json:"maxDownloadConcurrency" query:"maxDownloadConcurrency"`
	UserAgent                     string `form:"userAgent" json:"userAgent" query:"userAgent"`
}

var searchOptions = map[string]string{
	"itunes":       "iTunes",
	"podcastindex": "PodcastIndex",
}
var searchProvider = map[string]service.SearchService{
	"itunes":       new(service.ItunesService),
	"podcastindex": new(service.PodcastIndexService),
}

func AddPage(c *gin.Context) {
	setting := c.MustGet("setting").(*db.Setting)
	c.HTML(http.StatusOK, "addPodcast.html", gin.H{"title": "Add Podcast", "setting": setting, "searchOptions": searchOptions})
}

func HomePage(c *gin.Context) {

	podcasts, err := service.GetAllPodcasts("")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Failed to get podcasts.", "err": err})
		return
	}

	setting := c.MustGet("setting").(*db.Setting)
	c.HTML(http.StatusOK, "index.html", gin.H{"title": "Podgrab", "podcasts": podcasts, "setting": setting})
}

func PodcastPage(c *gin.Context) {
	var searchByIdQuery SearchByIdQuery
	if c.ShouldBindUri(&searchByIdQuery) == nil {

		var podcast db.Podcast

		if err := db.GetPodcastById(searchByIdQuery.Id, &podcast); err == nil {
			var pagination model.Pagination
			if c.ShouldBindQuery(&pagination) == nil {
				var page, count int
				if page = pagination.Page; page == 0 {
					page = 1
				}
				if count = pagination.Count; count == 0 {
					count = 10
				}
				setting := c.MustGet("setting").(*db.Setting)
				totalCount := len(podcast.PodcastItems)
				totalPages := int(math.Ceil(float64(totalCount) / float64(count)))
				nextPage, previousPage := 0, 0
				if page < totalPages {
					nextPage = page + 1
				}
				if page > 1 {
					previousPage = page - 1
				}

				from := (page - 1) * count
				to := page * count
				if to > totalCount {
					to = totalCount
				}
				c.HTML(http.StatusOK, "episodes.html", gin.H{
					"title":          podcast.Title,
					"podcastItems":   podcast.PodcastItems[from:to],
					"setting":        setting,
					"page":           page,
					"count":          count,
					"totalCount":     totalCount,
					"totalPages":     totalPages,
					"nextPage":       nextPage,
					"previousPage":   previousPage,
					"downloadedOnly": false,
					"podcastId":      searchByIdQuery.Id,
				})
			} else {
				c.JSON(http.StatusBadRequest, err)
			}
		} else {
			c.JSON(http.StatusBadRequest, err)
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
	}

}

func getItemsToPlay(itemIds []string, podcastId string, tagIds []string) ([]db.PodcastItem, error) {

	if len(itemIds) > 0 {

		toAdd, err := service.GetAllPodcastItemsByIds(itemIds)
		if err != nil {
			return nil, pkgErrors.Wrap(err, "failed to get podcast items")
		}

		return *toAdd, nil
	}

	if podcastId != "" {
		pod, err := service.GetPodcastById(podcastId)
		if err != nil {
			return nil, pkgErrors.Wrap(err, "failed to get podcast")
		}

		return pod.PodcastItems, nil
	}

	if len(tagIds) != 0 {

		tags, err := service.GetTagsByIds(tagIds)
		if err != nil {
			return nil, pkgErrors.Wrap(err, "failed to get tags")
		}

		var podIds []string

		for _, tag := range *tags {
			for _, pod := range tag.Podcasts {
				podIds = append(podIds, pod.ID)
			}
		}

		items, err := service.GetAllPodcastItemsByPodcastIds(podIds)
		if err != nil {
			return nil, pkgErrors.Wrap(err, "failed to get podcast items")
		}

		return *items, nil
	}

	return nil, nil
}

func PlayerPage(c *gin.Context) {

	itemIds, hasItemIds := c.GetQueryArray("itemIds")
	podcastId, hasPodcastId := c.GetQuery("podcastId")
	tagIds, hasTagIds := c.GetQueryArray("tagIds")

	title := "Podgrab"

	var items []db.PodcastItem
	var totalCount int64

	if hasItemIds {

		toAdd, err := service.GetAllPodcastItemsByIds(itemIds)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get podcast items", "err": err})
		}

		items = *toAdd
		totalCount = int64(len(items))

	} else if hasPodcastId {
		pod, err := service.GetPodcastById(podcastId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get podcast", "err": err})
			return
		}

		items = pod.PodcastItems
		title = "Playing: " + pod.Title
		totalCount = int64(len(items))

	} else if hasTagIds {

		tags, err := service.GetTagsByIds(tagIds)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get tags", "err": err})
			return
		}

		var tagNames []string
		var podIds []string
		for _, tag := range *tags {
			tagNames = append(tagNames, tag.Label)
			for _, pod := range tag.Podcasts {
				podIds = append(podIds, pod.ID)
			}
		}

		returned, err := service.GetAllPodcastItemsByPodcastIds(podIds)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get podcast items", "err": err})
			return
		}

		if len(tagNames) == 1 {
			title = fmt.Sprintf("Playing episodes with tag : %s", tagNames[0])
		} else {
			title = fmt.Sprintf("Playing episodes with tags : %s", strings.Join(tagNames, ", "))
		}

		items = *returned
	} else {
		title = "Playing Latest Episodes"
		if err := db.GetPaginatedPodcastItems(1, 20, nil, nil, time.Time{}, &items, &totalCount); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get paginated podcast items", "err": err})
			return
		}
	}
	setting := c.MustGet("setting").(*db.Setting)

	c.HTML(http.StatusOK, "player.html", gin.H{
		"title":          title,
		"podcastItems":   items,
		"setting":        setting,
		"count":          len(items),
		"totalCount":     totalCount,
		"downloadedOnly": true,
	})

}
func SettingsPage(c *gin.Context) {

	setting := c.MustGet("setting").(*db.Setting)
	diskStats, _ := db.GetPodcastEpisodeDiskStats()
	c.HTML(http.StatusOK, "settings.html", gin.H{
		"setting":   setting,
		"title":     "Update your preferences",
		"diskStats": diskStats,
	})

}
func BackupsPage(c *gin.Context) {

	files, err := service.GetAllBackupFiles()
	var allFiles []interface{}
	setting := c.MustGet("setting").(*db.Setting)

	for _, file := range files {
		arr := strings.Split(file, string(os.PathSeparator))
		name := arr[len(arr)-1]
		subsplit := strings.Split(name, "_")
		dateStr := subsplit[2]
		date, err := time.Parse("2006.01.02", dateStr)
		if err == nil {
			toAdd := map[string]interface{}{
				"date": date,
				"name": name,
				"path": strings.ReplaceAll(file, string(os.PathSeparator), "/"),
			}
			allFiles = append(allFiles, toAdd)
		}
	}

	if err == nil {
		c.HTML(http.StatusOK, "backups.html", gin.H{
			"backups": allFiles,
			"title":   "Backups",
			"setting": setting,
		})
	} else {
		c.JSON(http.StatusBadRequest, err)
	}

}

func getSortOptions() interface{} {
	return []struct {
		Label, Value string
	}{
		{"Release (asc)", "release_asc"},
		{"Release (desc)", "release_desc"},
		{"Duration (asc)", "duration_asc"},
		{"Duration (desc)", "duration_desc"},
	}
}
func AllEpisodesPage(c *gin.Context) {

	var filter model.EpisodesFilter
	err := c.ShouldBindQuery(&filter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "err": err})
		return
	}

	filter.VerifyPaginationValues()
	setting := c.MustGet("setting").(*db.Setting)

	podcasts, err := service.GetAllPodcasts("")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get podcasts", "err": err})
		return
	}

	tags, err := db.GetAllTags("")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get tags", "err": err})
		return
	}

	toReturn := gin.H{
		"title":        "All Episodes",
		"podcastItems": []db.PodcastItem{},
		"setting":      setting,
		"page":         filter.Page,
		"count":        filter.Count,
		"filter":       filter,
		"podcasts":     podcasts,
		"tags":         tags,
		"sortOptions":  getSortOptions(),
	}
	c.HTML(http.StatusOK, "episodes_new.html", toReturn)
}

func AllTagsPage(c *gin.Context) {

	var pagination model.Pagination
	var page, count int

	err := c.ShouldBindQuery(&pagination)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "err": err})
		return
	}

	if page = pagination.Page; page == 0 {
		page = 1
	}
	if count = pagination.Count; count == 0 {
		count = 10
	}

	var tags []db.Tag
	var totalCount int64

	err = db.GetPaginatedTags(page, count, &tags, &totalCount)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	setting := c.MustGet("setting").(*db.Setting)
	totalPages := math.Ceil(float64(totalCount) / float64(count))
	nextPage, previousPage := 0, 0
	if float64(page) < totalPages {
		nextPage = page + 1
	}
	if page > 1 {
		previousPage = page - 1
	}
	toReturn := gin.H{
		"title":        "Tags",
		"tags":         tags,
		"setting":      setting,
		"page":         page,
		"count":        count,
		"totalCount":   totalCount,
		"totalPages":   totalPages,
		"nextPage":     nextPage,
		"previousPage": previousPage,
	}
	c.HTML(http.StatusOK, "tags.html", toReturn)
}

func Search(c *gin.Context) {
	var searchQuery SearchGPodderData

	err := c.ShouldBindQuery(&searchQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "err": err})
		return
	}

	var searcher service.SearchService
	var isValidSearchProvider bool
	if searcher, isValidSearchProvider = searchProvider[searchQuery.SearchSource]; !isValidSearchProvider {
		searcher = new(service.PodcastIndexService)
	}

	data, err := searcher.Query(searchQuery.Q)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	allPodcasts, err := service.GetAllPodcasts("")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Failed to get podcasts", "err": err})
		return
	}

	urls := make(map[string]string, len(*allPodcasts))
	for _, pod := range *allPodcasts {
		urls[pod.URL] = pod.ID
	}
	for _, pod := range data {
		_, ok := urls[pod.URL]
		pod.AlreadySaved = ok
	}
	c.JSON(200, data)
}

func GetOmpl(c *gin.Context) {

	usePodgrabLink := c.DefaultQuery("usePodgrabLink", "false") == "true"

	data, err := service.ExportOmpl(usePodgrabLink, getBaseUrl(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=podgrab-export.opml")
	c.Data(200, "text/xml", data)
}
func UploadOpml(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	content := buf.String()
	err = service.AddOpml(content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	} else {
		c.JSON(200, gin.H{"success": "File uploaded"})
	}
}
