package item

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	logger "bitbucket.org/conorit/golib-logger"
	pat "github.com/gorilla/pat"
)

var (
	log = logger.New("item")
)

//Item interface for use in API
type Item interface {
	Validate() error
	Blank() interface{}                                //return empty item
	New(interface{}) (interface{}, error)              //return item with ID
	Get(ID string) (interface{}, error)                //return item with ID
	GetKey(key map[string]string) (interface{}, error) //return item with ID
	Upd(string, interface{}) error
	Del(ID string) error
}

//AddItemRoutes ...
func AddItemRoutes(r *pat.Router, item string, i Item) {
	log.Debug.Printf("Adding item")
	URLsimple := "/" + item
	URLwithID := "/" + item + "/{id}"
	itemType := reflect.TypeOf(i) //.Elem()

	log.Debug.Printf("Item type %s", reflect.TypeOf(i))

	//HTTP POST /item
	//with JSON body is used to create an item
	//on success the new item is echoed with an id
	r.Post(URLsimple, func(res http.ResponseWriter, req *http.Request) {
		//parse the JSON body into a new copy of the item type
		jsonDecoder := json.NewDecoder(req.Body)
		newItemPtr := reflect.New(itemType).Interface()
		if err := jsonDecoder.Decode(&newItemPtr); err != nil {
			http.Error(res, fmt.Sprintf("Invalid JSON for %s: %v", item, err.Error()), http.StatusBadRequest)
			return
		}
		log.Debug.Printf("Parsed JSON into %s: %+v", item, newItemPtr)

		//create the new item
		itemData, err := i.New(newItemPtr)
		if err != nil {
			http.Error(res, fmt.Sprintf("Cannot create %s: %v", item, err.Error()), http.StatusBadRequest)
			return
		}

		//succes: output Item
		if itemJSON, err := json.Marshal(itemData); err != nil {
			http.Error(res, fmt.Sprintf("Internal Error: %v", err.Error()), http.StatusServiceUnavailable)
		} else {
			res.Write([]byte(itemJSON))
		}
	}) //HTTP POST /item

	//r.Get(URLsimple, GetItemHandler) ...for list

	//HTTP GET /item/<id>
	r.Get(URLwithID, func(res http.ResponseWriter, req *http.Request) {
		ID := req.URL.Query().Get(":id")
		if itemData, err := i.Get(ID); err != nil {
			http.Error(res, fmt.Sprintf("Cannot get %s.id=%s: %v", item, ID, err), http.StatusNotFound)
		} else {
			if itemJSON, err := json.Marshal(itemData); err != nil {
				http.Error(res, fmt.Sprintf("Internal Error: %v", err.Error()), http.StatusServiceUnavailable)
			} else {
				//success: output item
				res.Write([]byte(itemJSON))
			}
		}
	}) //HTTP GET /item/<id>

	//HTTP GET /item (with search params in URL, e.g. ?email=a@b.c
	r.Get(URLsimple, func(res http.ResponseWriter, req *http.Request) {
		log.Debug.Printf("Getting")
		itemData := i.Blank()
		var err error

		//make search key with all URL parameters
		key := make(map[string]string)
		for n, v := range req.URL.Query() {
			log.Debug.Printf("query has %s=%+v", n, v)
			if len(v) == 1 {
				key[n] = v[0]
			}
		}
		if itemData, err = i.GetKey(key); err != nil {
			http.Error(res, fmt.Sprintf("Cannot get %s(%+v): %v", item, key, err), http.StatusNotFound)
		}

		if itemJSON, err := json.Marshal(itemData); err != nil {
			http.Error(res, fmt.Sprintf("Internal Error: %v", err.Error()), http.StatusServiceUnavailable)
		} else {
			//success: output item
			res.Write([]byte(itemJSON))
		}
	}) //HTTP GET /item/<id>

	//HTTP PUT /item/<id>
	//with JSON body is used to update an item
	//id in URL and body should match
	r.Put(URLwithID, func(res http.ResponseWriter, req *http.Request) {
		//parse the JSON body into a new copy of the item type
		jsonDecoder := json.NewDecoder(req.Body)
		newItemPtr := reflect.New(itemType).Interface()
		if err := jsonDecoder.Decode(&newItemPtr); err != nil {
			http.Error(res, fmt.Sprintf("Invalid JSON for %s: %v", item, err.Error()), http.StatusBadRequest)
			return
		}
		log.Debug.Printf("Parsed JSON into %s: %+v", item, newItemPtr)
		ID := req.URL.Query().Get(":id")
		var err error
		if i.Upd(ID, newItemPtr); err != nil {
			http.Error(res, fmt.Sprintf("Get %s.id=%s failed: %v", item, ID, err.Error()), http.StatusNotFound)
			return
		}
		log.Debug.Printf("Updated")
	}) //HTTP PUT /item/<id>

	//HTTP DELETE /item/<id>
	r.Delete(URLwithID, func(res http.ResponseWriter, req *http.Request) {
		ID := req.URL.Query().Get(":id")
		if err := i.Del(ID); err != nil {
			http.Error(res, fmt.Sprintf("Delete %s.id=%s failed: %v", item, ID, err.Error()), http.StatusNotFound)
			return
		}
		log.Debug.Printf("Deleted %s.id=%s", item, ID)
	}) //DELETE /item/<id>
}
