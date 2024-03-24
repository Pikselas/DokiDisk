package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Pikselas/Octodrive/Octo"
	"github.com/Pikselas/Octodrive/Octo/ToOcto"
)

var user, octo_err = ToOcto.NewOctoUser("", "")

func UploadFileToRemote(path string, octo_user *ToOcto.OctoUser, src io.Reader, closer io.Closer) error {

	fmt.Println(path)
	drive, octo_err := Octo.NewOctoDrive(octo_user, "_doki_drive")
	if octo_err != nil {
		return octo_err
	}

	file := drive.Create(src)
	for {
		err := file.WriteAll()
		if err == nil {
			break
		} else {
			for err != nil {
				err = file.RetryWriteChunk()
			}
		}
	}
	err := drive.Save(path, file)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("File uploaded")
	closer.Close()
	return nil
}

func StreamFile(of *Octo.OctoFile, Type string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", Type)
		w.Header().Set("Accept-Ranges", "bytes")
		byteRange := r.Header.Get("Range")
		parsedStart := int64(0)
		bSplit := strings.Split(byteRange, "=")
		if len(bSplit) == 2 {
			bSplit = strings.Split(bSplit[1], "-")
			if len(bSplit) == 2 {
				var err error
				parsedStart, err = strconv.ParseInt(bSplit[0], 10, 64)
				if err != nil {
					panic(err)
				}
				fmt.Println(parsedStart)
			}
			w.Header().Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", parsedStart, of.GetSize(), of.GetSize()))
			w.Header().Set("Content-Length", fmt.Sprint(int64(of.GetSize())-parsedStart))
			w.WriteHeader(http.StatusPartialContent)
		} else {
			w.Header().Set("Content-Length", fmt.Sprint(of.GetSize()))
		}
		fmt.Println("Getting", parsedStart, of.GetSize(), parsedStart)
		re, err := of.GetSeekReader()
		if err != nil {
			panic(err)
		}
		re.Seek(parsedStart, io.SeekStart)
		defer re.Close()
		fmt.Println("Sending")
		n, err := io.Copy(w, re)
		fmt.Println("Sent", n, err)
	}
}

func main() {

	if octo_err != nil {
		panic(octo_err)
	}

	http.HandleFunc("/upload_file", func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		pipe_reader, pipe_writer := io.Pipe()
		branched_reader := io.TeeReader(r.Body, pipe_writer)

		defer pipe_writer.Close()
		defer pipe_reader.Close()

		file_name := r.Header.Get("-file-name")
		go UploadFileToRemote(file_name, user, branched_reader, pipe_writer)

		file_name_und := strings.ReplaceAll(file_name, "/", "_")
		ffmpeg_cmd := exec.Command("./ffmpeg", "-i", "-",
			"-ss", "00:00:01",
			"-vframes", "1",
			"-y", file_name_und+".jpg")

		ffmpeg_cmd.Stdin = pipe_reader

		_, err := ffmpeg_cmd.Output()
		if err != nil {
			println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		io.Copy(io.Discard, pipe_reader)
		thumbnail_file, err := os.Open(file_name_und + ".jpg")
		if err != nil {
			println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer thumbnail_file.Close()
		octo_err = user.Transfer("_doki_drive", "thumbnails/"+file_name_und+".jpg", thumbnail_file)
		if octo_err != nil {
			println(octo_err.Error())
			http.Error(w, octo_err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/file_list/{file_path...}", func(w http.ResponseWriter, r *http.Request) {

		drive, err := Octo.NewOctoDrive(user, "_doki_drive")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		file_nav, err := drive.NewFileNavigator()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = file_nav.GotoDirectory(r.PathValue("file_path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}

		files := file_nav.GetItemList()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)

	})

	http.HandleFunc("/get_file/{file_path...}", func(w http.ResponseWriter, r *http.Request) {
		drive, err := Octo.NewOctoDrive(user, "_doki_drive")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		file, err := drive.Load(r.PathValue("file_path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		StreamFile(file, "application/octet-stream")(w, r)

	})

	http.HandleFunc("/get_thumbnail/{file_path...}", func(w http.ResponseWriter, r *http.Request) {

		reader, err := user.GetContent("_doki_drive", "thumbnails/"+strings.ReplaceAll(r.PathValue("file_path"), "/", "_")+".jpg")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer reader.Close()
		w.Header().Set("Content-Type", "image/jpeg")
		io.Copy(w, reader)
	})

	http.HandleFunc("/{file...}", func(w http.ResponseWriter, r *http.Request) {

		http.ServeFile(w, r, r.PathValue("file"))
	})

	http.ListenAndServe(":8080", nil)
}
