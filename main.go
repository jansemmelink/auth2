package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	logger "bitbucket.org/conorit/golib-logger"
	pidfile "bitbucket.org/conorit/golib-pidfile"
	"github.com/gorilla/mux"
	"github.com/gorilla/pat"
	"github.com/jansemmelink/auth2/auth"
	"github.com/jansemmelink/auth2/item"
)

var (
	log = logger.New("main")
)

func main() {
	// process command line options
	pidfile.WritePIDFile("/tmp/auth.pid")
	addrPtr := flag.String("addr", "localhost", "IP address to bind for HTTP")
	portPtr := flag.Int("port", 3000, "TCP Port to bind")
	debugBoolPtr := flag.Bool("d", false, "Debug")
	flag.Parse()
	if *debugBoolPtr {
		logger.SetDefaultLevel(logger.LevelDebug)
	}

	// start the http server
	addr := fmt.Sprintf("%s:%d", *addrPtr, *portPtr)
	log.Info.Printf("Listening on %s", addr)
	http.Handle("/", app())
	if err := http.ListenAndServe(addr, nil /*App()*/); err != nil {
		log.Error.Printf("Failed: %v", err)
		os.Exit(1)
	}
	log.Info.Printf("Terminated")
} /*main()*/

func app() http.Handler {
	r := pat.New()
	r.Options("/", corsHandler)
	auth.AddAuthRoutes(r)
	item.AddItemRoutes(r, "person", item.Person{})

	//debug output for all routes... later use to document
	r.Router.Walk(
		func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			tpl, _ := route.GetPathTemplate()
			met, _ := route.GetMethods()
			log.Debug.Printf("%v %v", tpl, met)
			return nil
		})
	return contentType(r)
}

func errorHandler(res http.ResponseWriter, req *http.Request, err string) {
	log.Info.Printf("ERROR Handler: %s", err)
	//generate error response
	res.Write([]byte("{\"code\":-1,\"desc\":\"Failed: " + err + "\"}"))
} //errorHandler()

func unknownHandler(res http.ResponseWriter, req *http.Request) {
	log.Info.Printf("Ignore unknown URI: %s", req.RequestURI)
	//generate error response
	http.Error(res, "{\"code\":-1,\"desc\":\"Unknown Resource\"}", http.StatusNotFound)
	return
} //unknownHandler()

/* CORS: Example HTTP Trace:
OPTIONS /users HTTP/1.1
Host: localhost:3000
Connection: keep-alive
Access-Control-Request-Method: POST
Origin: http://localhost:4200
User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36
Access-Control-Request-Headers: content-type
Accept: * / *	                                    <-- JS: no spacing in trace, added spacing not to be comment in golang
Referer: http://localhost:4200/register
Accept-Encoding: gzip, deflate, br
Accept-Language: en-GB,en-US;q=0.8,en;q=0.6

HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Content-Type: application/json
Content-Type: application/json
Date: Thu, 12 Oct 2017 05:22:13 GMT
Content-Length: 0
*/
func corsHandler(res http.ResponseWriter, req *http.Request) {
	log.Trace.Printf("CORS HANDLER...")
	res.Header().Add("Content-Type", "application/json")
	if origin := req.Header.Get("Origin"); origin != "" {
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST, GET, PUT, DELETE")
		res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token")
	}
} /*corsHandler()*/

// contentType is middleware that adds an application/json Content-Type header
// to all outgoing responses.
func contentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "application/json")

		//allowedHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization,X-CSRF-Token"
		if origin := req.Header.Get("Origin"); origin != "" {
			res.Header().Set("Access-Control-Allow-Origin", "*")
			//res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			//res.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			//res.Header().Set("Access-Control-Expose-Headers", "Authorization")
		}
		h.ServeHTTP(res, req)
	})
}

func urlParamInt(url *url.URL, name string, def, min, max int) (int, error) {
	v := url.Query().Get(name)
	if v == "" {
		return def, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Errorf("URL parameter %s='%s' must have integer value", name, v)
	}
	if i < min {
		return def, fmt.Errorf("URL parameter %s='%s' must have integer value >= %d", name, v, def)
	}
	if (max > min) && (i > max) {
		return def, fmt.Errorf("URL parameter %s='%s' must have integer value %d..%d", name, v, min, max)
	}
	return i, nil
} //urlParamInt()
