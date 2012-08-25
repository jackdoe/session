package session
import (
	"testing"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"	
	"fmt"
	"reflect"
	"os"
	"time"
	"encoding/gob"
)
func compare(a reflect.Value,b reflect.Value) bool {
	switch a.Kind() {
	case reflect.Slice,reflect.Array:
		if a.Len() != b.Len() { return false }
		for i := 0; i < a.Len(); i++ {
			if !compare(a.Index(i),b.Index(i)) { return false }
		}
		return true
	case reflect.Map:
		if a.Len() != b.Len() { return false }
   		for _, k := range a.MapKeys() {
   			if !compare(a.MapIndex(k),b.MapIndex(k)) { return false }
   		}
   		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() == b.Uint()
	case reflect.Bool:
		return a.Bool() == b.Bool()
	case reflect.Float32, reflect.Float64:
		return a.Float() == b.Float()
	case reflect.String:
		return a.String() == b.String()
	}
	return false
}
func TestNew(t *testing.T) {
	db, err := sql.Open("sqlite3", "./foo.db")
	var created_sessions int64 = 0
	if err != nil {
		t.Errorf("Failed to open database:", err)
		return
	}
	defer os.Remove("./foo.db")
	err = Init(db,"session")
	if err != nil {
		t.Errorf("Failed to insert record:", err)
		return
	}
	s := NewWithId("x") 
	if (s.Id == "x") {
		t.Errorf("id must not be 'x'")
	}
	gob.Register(map[string]int{})
	gob.Register(map[string]string{})
	gob.Register(map[int]string{})
	gob.Register(map[interface{}]string{})
	values := []interface{}{5,5.0,"x","§",[]string{"x§§RDd∂ßƒ¥˙∆©ˆø˙¨ˆå¥ß∂ˆ¨åß∂˙©","y"},-1,-0.1,1<<30, 
							map[string]int{"a":5},
							map[string]string{"x":"aaa","y":"bbbb"},
							map[int]string{5:"a",6:"n"},
							map[interface{}]string{5:"x","w":"x2",5.0:"five"},
							[]string{}} 
	c := func(x *SessionObject,set bool) {
		i := 0
		for _,v := range values {
			key := fmt.Sprintf("v_%d",i) 
			if set {
				x.Set(key,v)
			}
			_v,_ := x.Get(key);
			if !compare(reflect.ValueOf(_v),reflect.ValueOf(v)) {
				t.Errorf("%s: stored( %#v ) != extracted( %#v )",key,v,_v)
			}
			i++
		}
	}
	c(s,true)
	c(NewWithId(s.Id),false); created_sessions++
	b := NewWithId("x"); created_sessions++
	if (b.Id == s.Id) {
		t.Errorf("generated ID must not be the same as the old one")
	}
	for i:= 0; i< 10 ;i++ {
		_ = NewWithId("x")
		created_sessions++
	}
	CookieExpireInSeconds = 0
	time.Sleep(1000000000)
	expired := Expire()
	if expired != created_sessions {
		t.Errorf("expected to expire %d sessions but expired %d",created_sessions,expired)
	}
}
