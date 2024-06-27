package main

import (
	"fmt"
	"net/http"

	"github.com/Pikselas/Octodrive/Octo/ToOcto"
)

func CheckLoginThen(f func(http.ResponseWriter, *http.Request, string, string)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := r.Cookie("doki_user")
		if err != nil {
			// http.Error(w, "User email not Provided", http.StatusUnauthorized)
			// return
			http.Redirect(w, r, "/entry", http.StatusFound)
			return
		}
		token, err := r.Cookie("doki_pass")
		if err != nil {
			//http.Error(w, "User token not Provided", http.StatusUnauthorized)
			http.Redirect(w, r, "/entry", http.StatusFound)
			return
		}

		fmt.Println(user, token)

		f(w, r, user.Value, token.Value)
	}
}

func main() {

	http.HandleFunc("/upload_file", CheckLoginThen(Upload_file))
	http.HandleFunc("/get_file/{file_path...}", CheckLoginThen(Get_file_from_path))
	http.HandleFunc("/file_list/{file_path...}", CheckLoginThen(List_files))
	http.HandleFunc("/get_thumbnail/{file_path...}", CheckLoginThen(Get_file_thumbnail))

	// get shared link
	http.HandleFunc("/get_link/{file_path}", CheckLoginThen(Get_shared_link))
	// get shared file
	http.HandleFunc("/get/{file_key}", Get_shared_file)

	// http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
	// 	http.ServeFile(w, r, r.PathValue("file"))
	// })

	static_file_server := http.FileServer(http.Dir("./static"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/home", http.StatusFound)
		} else {
			static_file_server.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/home", CheckLoginThen(func(w http.ResponseWriter, r *http.Request, email string, token string) {
		http.ServeFile(w, r, "./static/home.html")
	}))
	http.HandleFunc("/entry", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/entry.html")
	})
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		Email := r.FormValue("email")
		Token := r.FormValue("password")

		_, err := ToOcto.NewOctoUser(Email, Token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "doki_user",
			Value: Email,
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "doki_pass",
			Value: Token,
		})
		//http.Redirect(w, r, "/index", http.StatusAccepted)
	})

	http.ListenAndServe(":8080", nil)
}
