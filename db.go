package gaia

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/data/builtin/mongo"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/routes"
	"github.com/elos/models"
	"golang.org/x/net/websocket"
)

type ChangeTransport struct {
	ChangeKind data.ChangeKind        `json:"change_kind"`
	RecordKind data.Kind              `json:"record_kind"`
	Record     map[string]interface{} `json:"record"`
}

func recordEndpoint(host string) string {
	return host + routes.Record
}

func recordQueryEndpoint(host string) string {
	return host + routes.RecordQuery
}

func recordChangesEndpoint(host string) string {
	return host + routes.RecordChanges
}

// DB implements the data.DB interface, and communicates over HTTP
// with the gaia server to complete it's actions
type DB struct {
	URL, Username, Password string
	*http.Client
}

func (db *DB) recordURL(v url.Values) string {
	return recordEndpoint(db.URL) + "?" + v.Encode()
}

func (db *DB) recordQueryURL(v url.Values) string {
	return recordQueryEndpoint(db.URL) + "?" + v.Encode()
}

func (db *DB) recordChangesURL(v url.Values) string {
	return strings.Replace(recordChangesEndpoint(db.URL), "http", "ws", 1) + "?" + v.Encode()
}

func (db *DB) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	return db.do(req)
}

func (db *DB) deleteReq(url string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, nil)

	if err != nil {
		return nil, err
	}

	return db.do(req)
}

func (db *DB) postJSON(url string, v interface{}) (*http.Response, error) {
	requestBody, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return db.post(url, bytes.NewBuffer(requestBody))
}

func (db *DB) post(url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	return db.do(req)
}

func (db *DB) do(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(db.Username, db.Password)
	return db.Client.Do(req)
}

func (db *DB) save(r data.Record) error {
	// setup Params
	url := db.recordURL(url.Values{
		"kind": []string{r.Kind().String()},
		"id":   []string{r.ID().String()},
	})

	resp, err := db.postJSON(url, r)
	if err != nil {
		log.Printf("gaia.(*DB).save Error: while making request: %s", err)
		return data.ErrNoConnection
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		log.Print("gaia.(*DB).save Error: malformed request")
		return data.ErrNoConnection
	case http.StatusInternalServerError:
		return data.ErrNoConnection
	case http.StatusUnauthorized:
		return data.ErrAccessDenial
	case http.StatusCreated:
		fallthrough
	case http.StatusOK:
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("gaia.(*DB).save Error: reading response body: %s", err)
			return data.ErrNoConnection
		}

		if err := json.Unmarshal(body, r); err != nil {
			log.Printf("gaia.(*DB).save Error: unmarshalling JSON into record: %s", err)
			return data.ErrNoConnection
		}
	default:
		log.Printf("gaia.(*DB).save Error: unexpected status code: %d", resp.StatusCode)
		return data.ErrNoConnection
	}

	return nil
}

func (db *DB) deleteRecord(r data.Record) error {
	url := db.recordURL(url.Values{
		"kind": []string{r.Kind().String()},
		"id":   []string{r.ID().String()},
	})

	resp, err := db.deleteReq(url)
	if err != nil {
		log.Printf("gaia.(*DB).save Error: while making request: %s", err)
		return data.ErrNoConnection
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		log.Print("gaia.(*DB).deleteRecord Error: malformed request")
		return data.ErrNoConnection
	case http.StatusInternalServerError:
		return data.ErrNoConnection
	case http.StatusUnauthorized:
		return data.ErrAccessDenial
	case http.StatusNotFound:
		return data.ErrNotFound
	case http.StatusNoContent:
		// pass, there is nothing to do, the delete has succeeded
	default:
		log.Printf("gaia.(*DB).deleteRecord Error: unexpected status code: %d", resp.StatusCode)
		return data.ErrNoConnection
	}

	return nil
}

func (db *DB) query(k data.Kind, attrs data.AttrMap) (data.Iterator, error) {
	url := db.recordQueryURL(url.Values{
		"kind": []string{k.String()},
	})

	resp, err := db.postJSON(url, attrs)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		log.Print("gaia db bad request")
		fallthrough
	case http.StatusInternalServerError:
		return nil, data.ErrNoConnection
	case http.StatusUnauthorized:
		return nil, data.ErrAccessDenial
	case http.StatusOK:
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var resultsUntyped []*data.AttrMap

		if err := json.Unmarshal(body, &resultsUntyped); err != nil {
			return nil, err
		}

		var results []data.Record

		for _, attrs := range resultsUntyped {
			r := models.ModelFor(k)

			if err := transfer.TransferAttrs(attrs, r); err != nil {
				return nil, err
			}

			results = append(results, r)
		}

		out := make(chan data.Record)

		// cache results in another goroutine
		go func() {
			// read them off
			for _, r := range results {
				out <- r
			}
			close(out)
		}()

		return mem.Iter(out), nil
	default:
		log.Printf("Unexpected status code: %d", resp.StatusCode)
		return nil, data.ErrNoConnection
	}

	return nil, nil
}

// --- Exported Interface {{{

func (db *DB) NewID() data.ID {
	return data.ID(mongo.NewObjectID().Hex())
}

func (db *DB) ParseID(s string) (data.ID, error) {
	oid, err := mongo.ParseObjectID(s)
	return data.ID(oid.Hex()), err
}

func (db *DB) Save(r data.Record) error {
	return db.save(r)
}

func (db *DB) Delete(r data.Record) error {
	return db.deleteRecord(r)
}

func (db *DB) PopulateByID(r data.Record) error {
	params := url.Values{}
	params.Set("kind", r.Kind().String())
	params.Set("id", r.ID().String())
	url := db.recordURL(params)

	resp, err := db.get(url)

	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		log.Print("gaia.(*DB).PopulateByID Bad Request Error")
		fallthrough
	case http.StatusInternalServerError:
		log.Print("gaia.(*DB).PopulateByID Internal Server Error")
		fallthrough
	case http.StatusUnauthorized:
		log.Print("gaia.(*DB).PopulateByID Unauthorized Server Error")
		fallthrough
	case -1: // catch
		return data.ErrAccessDenial
	case http.StatusNotFound:
		return data.ErrNotFound
	case http.StatusCreated:
	case http.StatusOK:
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return json.Unmarshal(body, r)
	default:
		log.Printf("Unexpected status code: %d", resp.StatusCode)
		return data.ErrNoConnection
	}

	return nil
}

func (db *DB) PopulateByField(field string, value interface{}, r data.Record) error {
	iter, err := db.query(r.Kind(), data.AttrMap{
		field: value,
	})

	if err != nil {
		return err
	}

	if !iter.Next(r) {
		return data.ErrNotFound
	}

	return iter.Close()
}

func (db *DB) Query(k data.Kind) data.Query {
	return &query{
		kind:  k,
		attrs: make(data.AttrMap),
		db:    db,
	}

}

type query struct {
	kind  data.Kind
	db    *DB
	attrs data.AttrMap
}

func (q *query) Execute() (data.Iterator, error) {
	return q.db.query(q.kind, q.attrs)
}

func (q *query) Skip(i int) data.Query {
	panic("")
}

func (q *query) Limit(i int) data.Query {
	panic("")
}

func (q *query) Batch(i int) data.Query {
	panic("")
}

func (q *query) Order(fields ...string) data.Query {
	panic("")
}

func (q *query) Select(attrs data.AttrMap) data.Query {
	for k, v := range attrs {
		q.attrs[k] = v
	}
	return q
}

func (db *DB) Changes() *chan *data.Change {
	ch := make(chan *data.Change)

	go func() {
		// setup Params
		wsURL := db.recordChangesURL(url.Values{
			"public":  []string{db.Username},
			"private": []string{db.Password},
		})

		ws, err := websocket.Dial(wsURL, "", db.URL)
		if err != nil {
			log.Print("FAILED TO CONNECT TO GAIA")
			close(ch)
			return
		}
		defer ws.Close()

		var change transfer.ChangeTransport
		for {
			if err := websocket.JSON.Receive(ws, &change); err != nil {
				if err == io.EOF {
					close(ch)
					return
				}
				close(ch)
				return
			}

			m := models.ModelFor(change.RecordKind)
			transfer.TransferAttrs(change.Record, m)

			ch <- data.NewChange(change.ChangeKind, m)
		}
	}()

	return &ch
}

// --- }}}
