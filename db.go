package gaia

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/data/builtin/mongo"
	"github.com/elos/data/transfer"
	"github.com/elos/models"
)

type DB struct {
	URL, Username, Password string
	*http.Client
}

func (db *DB) NewID() data.ID {
	return data.ID(mongo.NewObjectID().Hex())
}

func (db *DB) ParseID(s string) (data.ID, error) {
	oid, err := mongo.ParseObjectID(s)
	return data.ID(oid.Hex()), err
}

func (db *DB) Save(r data.Record) error {
	params := url.Values{}
	params.Set("kind", r.Kind().String())
	params.Set("id", r.ID().String())
	url := db.URL + "/record/?" + params.Encode()

	requestBody, err := json.Marshal(r)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.SetBasicAuth(db.Username, db.Password)

	resp, err := db.Client.Do(req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
	case http.StatusInternalServerError:
		return data.ErrNoConnection
	case http.StatusUnauthorized:
		return data.ErrAccessDenial
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

func (db *DB) Delete(r data.Record) error {
	params := url.Values{}
	params.Set("kind", r.Kind().String())
	params.Set("id", r.ID().String())
	url := db.URL + "/record/?" + params.Encode()

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(db.Username, db.Password)

	resp, err := db.Client.Do(req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
	case http.StatusInternalServerError:
		return data.ErrNoConnection
	case http.StatusUnauthorized:
		return data.ErrAccessDenial
	case http.StatusNotFound:
		return data.ErrNotFound
	case http.StatusNoContent:
		return nil
	default:
		log.Printf("Unexpected status code: %d", resp.StatusCode)
		return data.ErrNoConnection
	}

	return nil
}

func (db *DB) PopulateByID(r data.Record) error {
	params := url.Values{}
	params.Set("kind", r.Kind().String())
	params.Set("id", r.ID().String())
	url := db.URL + "/record/?" + params.Encode()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(db.Username, db.Password)

	resp, err := db.Client.Do(req)
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
	params := url.Values{}
	params.Set("kind", r.Kind().String())
	url := db.URL + "/record/query/?" + params.Encode()

	requestBody, err := json.Marshal(data.AttrMap{
		field: value,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.SetBasicAuth(db.Username, db.Password)

	resp, err := db.Client.Do(req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
	case http.StatusInternalServerError:
	case http.StatusUnauthorized:
		return data.ErrAccessDenial
	case http.StatusOK:
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		results := make([]data.Record, 0)

		if err := json.Unmarshal(body, &results); err != nil {
			return err
		}

		if len(results) == 0 {
			return data.ErrNotFound
		} else {
			return transfer.TransferAttrs(results[0], r)
		}
	default:
		log.Printf("Unexpected status code: %d", resp.StatusCode)
		return data.ErrNoConnection
	}

	return nil
}

func (db *DB) Query(k data.Kind) data.Query {
	return &query{
		kind:  k,
		attrs: make(data.AttrMap),
		db:    db,
	}

}

func (db *DB) query(k data.Kind, attrs data.AttrMap) (data.Iterator, error) {
	params := url.Values{}
	params.Set("kind", k.String())
	url := db.URL + "/record/query/?" + params.Encode()

	requestBody, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(db.Username, db.Password)

	resp, err := db.Client.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
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

		go func() {
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

func (q *query) Select(attrs data.AttrMap) data.Query {
	for k, v := range attrs {
		q.attrs[k] = v
	}
	return q
}

func (db *DB) Changes() *chan *data.Change {
	panic("")
}
