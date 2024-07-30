package service

import (
	"archive/tar"
	"compress/gzip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/akhilrex/podgrab/db"
	"github.com/akhilrex/podgrab/internal/sanitize"
	"github.com/gobeam/stringy"
	pkgErrors "github.com/pkg/errors"
)

const (
	fileExtensionMp3 = ".mp3"
	fileExtensionJpg = ".jpg"
)

func Download(link string, episodeTitle string, podcastName string, prefix string) (string, error) {

	if link == "" {
		return "", errors.New("download path empty")
	}

	client := httpClient()

	req, err := createGetRequest(link)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to get response")
	}

	fileName, err := generateFileName(link, episodeTitle, fileExtensionMp3)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to get file name")
	}

	if prefix != "" {
		fileName = fmt.Sprintf("%s-%s", prefix, fileName)
	}

	folder, err := createDataFolderIfNotExists(podcastName)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create data folder")
	}

	finalPath := path.Join(folder, fileName)

	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		err = changeOwnership(finalPath)
		if err != nil {
			return "", pkgErrors.Wrap(err, "failed to change ownership")
		}
		return finalPath, nil
	}

	file, err := os.Create(finalPath)
	if err != nil {
		Logger.Errorw("Error creating file"+link, err)
		return "", err
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to save file")
	}
	defer file.Close()

	err = changeOwnership(finalPath)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to change ownership")
	}

	return finalPath, nil

}

func GetPodcastLocalImagePath(link string, podcastName string) (string, error) {

	fileName, err := generateFileName(link, "folder", fileExtensionJpg)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to get file name")
	}

	folder, err := createDataFolderIfNotExists(podcastName)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create data folder")
	}

	finalPath := path.Join(folder, fileName)
	return finalPath, nil
}

func CreateNfoFile(podcast *db.Podcast) error {

	fileName := "album.nfo"

	folder, err := createDataFolderIfNotExists(podcast.Title)
	if err != nil {
		return pkgErrors.Wrap(err, "failed to create data folder")
	}

	finalPath := path.Join(folder, fileName)

	type NFO struct {
		XMLName xml.Name `xml:"album"`
		Title   string   `xml:"title"`
		Type    string   `xml:"type"`
		Thumb   string   `xml:"thumb"`
	}

	toSave := NFO{
		Title: podcast.Title,
		Type:  "Broadcast",
		Thumb: podcast.Image,
	}
	out, err := xml.MarshalIndent(toSave, " ", "  ")
	if err != nil {
		return err
	}
	toPersist := xml.Header + string(out)
	return os.WriteFile(finalPath, []byte(toPersist), 0644)
}

func DownloadPodcastCoverImage(link string, podcastName string) (string, error) {
	if link == "" {
		return "", errors.New("download path empty")
	}
	client := httpClient()
	req, err := createGetRequest(link)
	if err != nil {
		Logger.Errorw("Error creating request: "+link, err)
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to get response: "+link)
	}
	defer resp.Body.Close()

	fileName, err := generateFileName(link, "folder", fileExtensionJpg)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to get file name")
	}

	folder, err := createDataFolderIfNotExists(podcastName)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create data folder")
	}

	finalPath := path.Join(folder, fileName)
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		err = changeOwnership(finalPath)
		if err != nil {
			return "", pkgErrors.Wrap(err, "failed to change ownership")
		}
		return finalPath, nil
	}

	file, err := os.Create(finalPath)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create file")
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to save file")
	}

	err = changeOwnership(finalPath)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to change ownership")
	}

	return finalPath, nil
}

func DownloadImage(link string, episodeId string, podcastName string) (string, error) {
	if link == "" {
		return "", errors.New("download path empty")
	}
	client := httpClient()
	req, err := createGetRequest(link)
	if err != nil {
		Logger.Errorw("Error creating request: "+link, err)
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		Logger.Errorw("Error getting response: "+link, err)
		return "", err
	}
	defer resp.Body.Close()

	fileName, err := generateFileName(link, episodeId, fileExtensionJpg)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to get file name: "+link)
	}

	folder, err := createDataFolderIfNotExists(podcastName)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create data folder")
	}

	imageFolder, err := createFolder("images", folder)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create image folder")
	}

	finalPath := path.Join(imageFolder, fileName)

	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		err = changeOwnership(finalPath)
		if err != nil {
			return "", pkgErrors.Wrap(err, "failed to change ownership")
		}
		return finalPath, nil
	}

	file, err := os.Create(finalPath)
	if err != nil {
		Logger.Errorw("Error creating file"+link, err)
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		Logger.Errorw("Error saving file"+link, err)
		return "", err
	}

	err = changeOwnership(finalPath)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to change ownership")
	}

	return finalPath, nil

}
func changeOwnership(path string) error {

	uid, err1 := strconv.Atoi(os.Getenv("PUID"))
	if err1 != nil {
		return pkgErrors.Wrap(err1, "failed to get PUID")
	}

	gid, err2 := strconv.Atoi(os.Getenv("PGID"))
	if err2 != nil {
		return pkgErrors.Wrap(err2, "failed to get PGID")
	}

	err := os.Chown(path, uid, gid)
	if err != nil {
		return pkgErrors.Wrap(err, "failed to change ownership")
	}

	return nil
}

func DeleteFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(filePath); err != nil {
		return err
	}
	return nil
}
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil

}

func GetAllBackupFiles() ([]string, error) {

	var files []string

	folder, err := createConfigFolderIfNotExists("backups")
	if err != nil {
		return nil, pkgErrors.Wrap(err, "failed to create backup")
	}

	err = filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files, err
}

func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func deleteOldBackup() error {
	files, err := GetAllBackupFiles()
	if err != nil {
		return pkgErrors.Wrap(err, "failed to get backup files")
	}

	if len(files) <= 5 {
		return nil
	}

	toDelete := files[5:]
	for _, file := range toDelete {
		err := DeleteFile(file)
		if err != nil {
			return pkgErrors.Wrap(err, "failed to delete old backup file")
		}
	}

	return nil
}

func GetFileSizeFromUrl(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}

	// Is our request ok?

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("did not receive 200")
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return 0, err
	}

	return int64(size), nil
}

func CreateBackup() (string, error) {

	backupFileName := "podgrab_backup_" + time.Now().Format("2006.01.02_150405") + ".tar.gz"

	folder, err := createConfigFolderIfNotExists("backups")
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to create backup folder")
	}

	configPath := os.Getenv("CONFIG")
	tarballFilePath := path.Join(folder, backupFileName)

	file, err := os.Create(tarballFilePath)
	if err != nil {
		return "", fmt.Errorf("could not create tarball file '%s', got error '%s'", tarballFilePath, err.Error())
	}
	defer file.Close()

	dbPath := path.Join(configPath, "podgrab.db")
	_, err = os.Stat(dbPath)
	if err != nil {
		return "", fmt.Errorf("could not find db file '%s', got error '%s'", dbPath, err.Error())
	}
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = addFileToTarWriter(dbPath, tarWriter)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to add db file to tarball")
	}

	err = deleteOldBackup()
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to delete old backup")
	}

	return backupFileName, nil
}

func addFileToTarWriter(filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file '%s', got error '%s'", filePath, err.Error())
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("could not get stat for file '%s', got error '%s'", filePath, err.Error())
	}

	header := &tar.Header{
		Name:    filePath,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("could not write header for file '%s', got error '%s'", filePath, err.Error())
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("could not copy the file '%s' data to the tarball, got error '%s'", filePath, err.Error())
	}

	return nil
}

func httpClient() *http.Client {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			//	r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	return &client
}

// createGetRequest creates an HTTP GET request for the specified URL.
// It also sets a custom User-Agent header if it is defined in the settings.
func createGetRequest(url string) (*http.Request, error) {

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "failed to create request")
	}

	setting := db.GetOrCreateSetting()
	if len(setting.UserAgent) > 0 {
		req.Header.Add("User-Agent", setting.UserAgent)
	}

	return req, nil
}

// createFolder creates a sanitized folder within a specified parent directory.
// It checks if the folder already exists, and if not, it creates the folder
// and changes its ownership.
func createFolder(folder string, parent string) (string, error) {

	folder = cleanFileName(folder)
	folderPath := path.Join(parent, folder)

	_, err := os.Stat(folderPath)
	if err != nil {

		if os.IsNotExist(err) {
			err = os.MkdirAll(folderPath, 0777)
			if err != nil {
				return "", pkgErrors.Wrap(err, "failed to create folder")
			}

			err = changeOwnership(folderPath)
			if err != nil {
				return "", pkgErrors.Wrap(err, "failed to change ownership")
			}
		}
	}

	return folderPath, nil
}

func createDataFolderIfNotExists(folder string) (string, error) {
	dataPath := os.Getenv("DATA")
	return createFolder(folder, dataPath)
}
func createConfigFolderIfNotExists(folder string) (string, error) {
	dataPath := os.Getenv("CONFIG")
	return createFolder(folder, dataPath)
}

func deletePodcastFolder(folder string) error {

	folder, err := createDataFolderIfNotExists(folder)
	if err != nil {
		return pkgErrors.Wrap(err, "failed to create data folder")
	}

	return os.RemoveAll(folder)
}

// generateFileName generates a sanitized file name based on the provided link, title,
// and default extension. It parses the URL to extract the file path and extension,
// and if no extension is found, it uses the default extension. The title is sanitized
// and converted to kebab-case to form the final file name.
func generateFileName(link string, title string, defaultExtension string) (string, error) {

	fileUrl, err := url.Parse(link)
	if err != nil {
		return "", pkgErrors.Wrap(err, "failed to parse url")
	}

	parsed := fileUrl.Path
	ext := filepath.Ext(parsed)

	if len(ext) == 0 {
		ext = defaultExtension
	}

	str := stringy.New(cleanFileName(title))
	return str.KebabCase().Get() + ext, nil
}

func cleanFileName(original string) string {
	return sanitize.Name(original)
}
