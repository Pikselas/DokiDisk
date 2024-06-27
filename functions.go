package main

import (
	"bytes"
	"encoding/base64"
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

func UploadFileToRemote(path string, octo_user *ToOcto.OctoUser, src io.Reader, closer io.Closer) error {

	fmt.Println(path)
	drive, octo_err := Octo.NewOctoDrive(octo_user, "_doki_drive")
	if octo_err != nil {
		return octo_err
	}

	file := drive.Create()
	file_write_data, err := Octo.InitializeFileWrite(file, src)
	if err != nil {
		return err
	}
	err = file.WriteAll(file_write_data)
	err_save := drive.Save(path, file)
	if err != nil {
		return err
	}
	if err_save != nil {
		fmt.Println(err_save.Error())
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

func Upload_file(w http.ResponseWriter, r *http.Request, user_email string, user_token string) {

	defer r.Body.Close()

	pipe_reader, pipe_writer := io.Pipe()
	branched_reader := io.TeeReader(r.Body, pipe_writer)

	defer pipe_writer.Close()
	defer pipe_reader.Close()

	file_name := r.Header.Get("-file-name")
	user, Err := ToOcto.NewOctoUser(user_email, user_token)
	if Err != nil {
		http.Error(w, Err.Error(), http.StatusInternalServerError)
		return
	}
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
	defer os.Remove(file_name_und + ".jpg")
	defer thumbnail_file.Close()

	octo_err := user.Transfer("_doki_drive", "thumbnails/"+file_name_und+".jpg", thumbnail_file)
	if octo_err != nil {
		println(octo_err.Error())
		http.Error(w, octo_err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func List_files(w http.ResponseWriter, r *http.Request, user_email string, user_token string) {
	user, Err := ToOcto.NewOctoUser(user_email, user_token)
	if Err != nil {
		http.Error(w, Err.Error(), http.StatusInternalServerError)
		return
	}
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
}
func Get_file(w http.ResponseWriter, r *http.Request, user_email string, user_token string, file_path string) {
	user, Err := ToOcto.NewOctoUser(user_email, user_token)
	if Err != nil {
		http.Error(w, Err.Error(), http.StatusInternalServerError)
		return
	}
	drive, err := Octo.NewOctoDrive(user, "_doki_drive")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file, err := drive.Load(file_path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	StreamFile(file, "application/octet-stream")(w, r)
}
func Get_file_from_path(w http.ResponseWriter, r *http.Request, user_email string, user_token string) {
	Get_file(w, r, user_email, user_token, r.PathValue("file_path"))
}
func Get_file_thumbnail(w http.ResponseWriter, r *http.Request, user_email string, user_token string) {

	user, Err := ToOcto.NewOctoUser(user_email, user_token)
	if Err != nil {
		http.Error(w, Err.Error(), http.StatusInternalServerError)
		return
	}
	reader, err := user.GetContent("_doki_drive", "thumbnails/"+strings.ReplaceAll(r.PathValue("file_path"), "/", "_")+".jpg")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", "image/jpeg")
	io.Copy(w, reader)
}

// -----------

type SharedFile struct {
	Path  string
	Email string
	Token string
}

var enc_dec, err = Octo.NewAesEncDec()

func Get_shared_link(w http.ResponseWriter, r *http.Request, user_email string, user_token string) {
	shared_file := SharedFile{Path: r.PathValue("file_path"), Email: user_email, Token: user_token}
	shared_file_json, err := json.Marshal(shared_file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//enc_data
	enc_data, err := enc_dec.Encrypt(bytes.NewReader(shared_file_json))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	// convert to base64
	base64_data := base64.NewEncoder(base64.URLEncoding, w)
	io.Copy(base64_data, enc_data)
	base64_data.Close()
}

func Get_shared_file(w http.ResponseWriter, r *http.Request) {
	file_key := r.PathValue("file_key")
	//dec_data
	dec_data := make([]byte, base64.URLEncoding.DecodedLen(len(file_key)))
	_, err := base64.URLEncoding.Decode(dec_data, []byte(file_key))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dec_reader, err := enc_dec.Decrypt(bytes.NewReader(dec_data))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	shared_file := SharedFile{}
	err = json.NewDecoder(dec_reader).Decode(&shared_file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println(shared_file)

	// get file
	Get_file(w, r, shared_file.Email, shared_file.Token, shared_file.Path)
}
