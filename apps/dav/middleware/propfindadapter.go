package dav

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func PropFindAdapter(handler http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Method: ", r.Method)

		if r.Method != "PROPFIND" {
			handler.ServeHTTP(w, r)
			return
		}

		log.Debug("request body: ", r.Body)
		log.Debug("request headers: ", r.Header)
		rh := NewResponseHijacker(w)
		handler.ServeHTTP(rh, r)

		// xmlbody := newDavXml()
		// xml.NewDecoder(bytes.NewReader(rh.body)).Decode(&xmlbody)s
		// log.Debug(rh.headers)
		// log.Debug(string(rh.body))
		// // log.Debug(xmlbody)
		// log.Debug(rh.status)

		// xmldoc := etree.NewDocument()
		// err := xmldoc.ReadFromBytes(rh.body)
		// if err != nil {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	log.Error(err)
		// 	return
		// }
		//
		// multistatusElement := xmldoc.SelectElement("multistatus")
		// responses := multistatusElement[0].ChildElements()
		// for _, resp := range responses {
		// 	log.Warn(resp)
		// 	respChildren := resp.ChildElements()
		// 	for _, respChild := range respChildren {
		// 		log.Warn(respChild)
		// 	}
		// }

		for key, valuemap := range rh.headers {
			w.Header().Set(key, strings.Join(valuemap, " "))
		}

		w.WriteHeader(rh.status)
		w.Write(rh.body)
	})
}
