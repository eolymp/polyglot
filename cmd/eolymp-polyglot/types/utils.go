package types

import (
	"context"
	"fmt"
	"github.com/eolymp/contracts/go/eolymp/keeper"
	"github.com/eolymp/contracts/go/eolymp/typewriter"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const RepeatNumber = 10
const TimeSleep = 10 * time.Second

func UpdateContentWithPictures(ctx context.Context, tw *typewriter.TypewriterService, content, source string) (string, error) {
	exts := []string{".png", ".jpeg", ".jpg"}
	files := FindFilesWithExtension(source, exts)
	for _, file := range files {
		if strings.Contains(content, file) {
			data, err := ioutil.ReadFile(source + file)
			if err != nil {
				log.Println("Failed to read file " + file)
				return "", err
			}
			var output *typewriter.UploadAssetOutput
			for i := 0; i < RepeatNumber; i++ {
				output, err = tw.UploadAsset(ctx, &typewriter.UploadAssetInput{Filename: file, Data: data})
				if err == nil {
					break
				}
				log.Println("Error while uploading asset")
			}
			if err != nil {
				log.Println("Error while uploading asset")
				return "", err
			}
			content = strings.ReplaceAll(content, file, output.Link)
		}
	}
	return content, nil
}

func FindFilesWithExtension(path string, exts []string) []string {
	var files []string
	_ = filepath.Walk(path, func(path string, f os.FileInfo, _ error) error {
		for _, ext := range exts {
			r, err := regexp.Match(ext, []byte(f.Name()))
			if err == nil && r {
				files = append(files, f.Name())
			}
		}
		return nil
	})
	return files
}

func MakeLocale(lang string) (string, error) {
	switch lang {
	case "ukrainian", "russian", "english", "hungarian":
		return lang[:2], nil
	default:
		return lang, fmt.Errorf("unknown language %#v", lang)
	}
}

func MakeObject(path string, kpr *keeper.KeeperService) (key string, err error) {

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return MakeObjectByData(data, kpr)
}

func MakeObjectByData(data []byte, kpr *keeper.KeeperService) (key string, err error) {
	var out *keeper.CreateObjectOutput
	for i := 0; i < RepeatNumber; i++ {
		out, err = kpr.CreateObject(context.Background(), &keeper.CreateObjectInput{Data: data})
		if err == nil {
			return out.Key, nil
		}

		log.Printf("Error while uploading file: %v", err)
		time.Sleep(TimeSleep)
	}

	return "", err
}

func RemoveSpaces(data string) string {
	l := 0
	r := len(data)
	for l < r && (data[l] == ' ' || data[l] == '\n') {
		l++
	}
	for l < r && (data[r-1] == ' ' || data[r-1] == '\n') {
		r--
	}
	return data[l:r]
}
