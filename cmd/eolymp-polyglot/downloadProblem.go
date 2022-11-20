package main

import (
	"compress/flate"
	"errors"
	"fmt"
	"github.com/mholt/archiver"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const DownloadsDir = "downloads"

func UpdateProblem(link string) error {
	data := GetData()
	inter, ok := data[link]
	if !ok {
		return errors.New("not found link in data")
	}
	pid := fmt.Sprintf("%v", inter)
	return DownloadAndImportProblem(link, &pid)
}

func DownloadAndImportProblem(link string, pid *string) error {
	path, err := DownloadProblem(link)
	if err != nil {
		return errors.New("failed to download problem")
	}

	err = ImportProblem(path, pid, false)

	data := GetData()
	data[link] = *pid
	SaveData(data)

	return err
}

func DownloadProblem(link string) (path string, err error) {
	log.Println("Started polygon download")
	if conf.Polygon.Login == "" || conf.Polygon.Password == "" {
		return "", fmt.Errorf("no polygon credentials")
	}
	if _, err := os.Stat(DownloadsDir); os.IsNotExist(err) {
		err = os.Mkdir(DownloadsDir, 0777)
		if err != nil {
			log.Println("Failed to create dir")
			return "", err
		}
	}
	name := link[strings.LastIndex(link, "/")+1:]
	location := DownloadsDir + "/" + name
	if err := DownloadFileAndUnzip(link, conf.Polygon.Login, conf.Polygon.Password, location); err != nil {
		log.Println("Failed to download from polygon")
		return "", err
	}

	log.Println("Finished polygon download")
	return location, nil
}

func DownloadFileAndUnzip(URL, login, password, location string) error {
	response, err := http.PostForm(URL, url.Values{"login": {login}, "password": {password}, "type": {"windows"}})
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != 200 {
		return errors.New("non 200 status code")
	}

	file, err := os.Create(location + ".zip")
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err = io.Copy(file, response.Body); err != nil {
		return err
	}

	if _, err := os.Stat(location); !os.IsNotExist(err) {
		if err = os.RemoveAll(location); err != nil {
			return err
		}
	}

	if err = os.Mkdir(location, 0777); err != nil {
		return err
	}

	z := archiver.Zip{
		CompressionLevel:       flate.DefaultCompression,
		MkdirAll:               true,
		SelectiveCompression:   true,
		ContinueOnError:        true,
		OverwriteExisting:      true,
		ImplicitTopLevelFolder: false,
	}
	if err = z.Unarchive(location+".zip", location); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
