package main
import (
	"github.com/jackdoe/session"
	"fmt"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
)
func main() {
	db, _ := sql.Open("sqlite3", "./foo.db")
	defer db.Close()
	session.Init(db,"session")
	session.CookieKey = "go.is.awesome"
	session.CookieDomain = "localhost"
	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		s := session.New(w,r)
		s.Set("list_of_numbers",[]int{1,2,3,4})
		fmt.Fprintf(w,"stored key list_of_numbers in session: %s",s.Id)
	})
	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		s := session.New(w,r)
		x,_ := s.Get("list_of_numbers")
		fmt.Fprintf(w,"extracted: %#v for key: list_of_numbers in session: %s",x,s.Id)
	})
	http.ListenAndServe("localhost:8080", nil)
}