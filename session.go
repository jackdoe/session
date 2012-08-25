// package session provides simple http.Cookie based session management with database/sql storage
package session
// to install:
// 		$ go get github.com/jackdoe/session
// to test:
// 		$ go test github.com/jackdoe/session # uses sqlite3 from github.com/mattn/go-sqlite3
// doc:
//		$ godoc github.com/jackdoe/session
//		http://go.pkgdoc.org/github.com/jackdoe/session
import (
	"fmt"
	"time"
	"encoding/gob"
	"encoding/base64"
	"bytes"
	"crypto/rand"
	"database/sql"	
	"net/http"
)
var (
	CookieKey string = "go.session"
	CookieValueLen int = 254
	CookieExpireInSeconds int = 60
	CookieSecure bool = false
	CookieHttpOnly bool = false
	CookieDomain string = "localhost"
	CookiePath string = "/"
	db *sql.DB
	table string
)
type SessionObject struct {
	Id string // unique idenfifier with CookieValueLen len
	Data map[string]interface{} // actual data
	Stamp int64 // last-upate time stamp
}


// session.Init() with sql.DB and table name, 
// returns error if it is unable to create the session table
// 
// example:
// 		db, _ := sql.Open("sqlite3", "./foo.db")
//		// or
//		db, _ := sql.Open("mysql", "user:pass@tcp(192.168.0.1:3306)/app")
//		err = session.Init(db,"session")

func Init(_db *sql.DB,_table string) (error){
	db = _db
	table = _table
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (id varchar(%d) UNIQUE, data blob, stamp bigint)",table,CookieValueLen + 1)
	_,err := db.Exec(query); if err != nil {
		fmt.Printf("failed to create table %s: %s",table, err.Error())
	}
	return err
}


/*
example:

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

extracts cookie value from http.Request.Cookie(CookieKey)
then creates new session object with that cookie value using NewWithId(ident)
finally it stores that id in http.ResponseWriter
*/
func New(w http.ResponseWriter, r *http.Request) (s *SessionObject) {
	cookie,_ := r.Cookie(CookieKey)
	var ident string
	if cookie != nil { ident = cookie.Value }
	s = find_or_create(ident)
	t := time.Now().Add(time.Duration(CookieExpireInSeconds) * time.Second)
	cookie = &http.Cookie {	CookieKey,s.Id,CookiePath,CookieDomain,t,t.String(),CookieExpireInSeconds,CookieSecure,CookieHttpOnly,s.Id,make([]string,0)}
	http.SetCookie(w,cookie)
	return 
}

// find_or_create by ID (requested id must be of len CookieValueLen)
func find_or_create(id string) (*SessionObject) {
	this := &SessionObject{id,make(map[string]interface{}),time.Now().Unix()}
	if len(this.Id) != CookieValueLen {
	    b := make([]byte, CookieValueLen * 2) 
	    rand.Read(b)
	    this.Id = base64.StdEncoding.EncodeToString(b)[:CookieValueLen]
		this.store()
	}
	var data []byte
	/* XXX: there can be user-modified cookie value with len == CookieValueLen, but the value is random anyway */
	query := fmt.Sprintf("SELECT data,stamp FROM `%s` WHERE id=? AND stamp > %d",table,time.Now().Unix() - int64(CookieExpireInSeconds))
	err := db.QueryRow(query,this.Id).Scan(&data,&this.Stamp); if err != nil {
		this.store()
	}
	err = deserialize(data,&this.Data); if err != nil {
		this.Data = make(map[string]interface{})
	}
	return this
}

// session.Expire() returns the number of expired sessions
func Expire() (int64) {
	query := fmt.Sprintf("DELETE FROM `%s` WHERE stamp < %d",table,time.Now().Unix() - int64(CookieExpireInSeconds))
	res,_ := db.Exec(query)
	rows,err := res.RowsAffected(); if err != nil {
		return 0
	}
	return rows
}

// the key must be a string
// uses encoding.gob, so you might have to do gob.Register on some types
// for example:
// 		gob.Register(map[interface{}]string{})
// now you can set:
// 		s.Set("x",map[interface{}]string{5:"x","w":"x2",5.0:"five"})
// example: 
// 		s.Set("list_of_numbers",[]int{1,2,3,4})
func (this *SessionObject) Set(k string, v interface{}) (*SessionObject){
	this.Data[k] = v
	this.store()
	return this
}
// returns true/false if key exists in the session or not
func (this *SessionObject) Has(k string) (bool) {
	_,ok := this.Get(k)
	return ok
}
// returns value of type interface{} and true/false for key existance
func (this *SessionObject) Get(k string) (interface {},bool) {
	v,ok := this.Data[k]
	return v,ok
}

func (this *SessionObject) store() {
	this.Stamp = time.Now().Unix()
	b,err := serialize(this.Data);
	if err != nil {
		this.err(err)
	}
	_,err = db.Exec("REPLACE INTO `" + table + "`(id,data,stamp) VALUES(?,?,?)",this.Id,b,this.Stamp); if err != nil {
		this.err(err)
	}
}

func (this *SessionObject) err(err error) {
	fmt.Printf("error: %s: %s\n",this.Id,err.Error())
}

// Copyright 2012 The Gorilla Authors. All rights reserved.
// serialize encodes a value using gob.
func serialize(src interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(src); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// deserialize decodes a value using gob.
func deserialize(src []byte, dst interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(src))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}
