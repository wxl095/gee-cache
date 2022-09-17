package cache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	m.Run()
}

func TestHTTP_Get(t *testing.T) {
	addr := "localhost:9999"
	ts := httptest.NewServer(NewHTTPPool(addr))
	log.Println("geecache is running at", ts.URL)
	defer ts.Close()
	path := ts.URL + defaultBasePath

	tests := []struct {
		name string
		url  string
		want struct {
			statusCode int
			body       []byte
		}
		wantErr bool
	}{
		{
			name: "No group name and key",
			url:  path,
			want: struct {
				statusCode int
				body       []byte
			}{statusCode: http.StatusBadRequest, body: []byte(http.StatusText(http.StatusBadRequest) + "\n")},
			wantErr: false,
		},
		{
			name: "The group does not exist",
			url:  path + "/test",
			want: struct {
				statusCode int
				body       []byte
			}{statusCode: http.StatusNotFound, body: []byte(http.StatusText(http.StatusNotFound) + "\n")},
			wantErr: false,
		},
		{
			name: "Hit the cache",
			url:  path + "scores/Tom",
			want: struct {
				statusCode int
				body       []byte
			}{statusCode: http.StatusOK, body: []byte("630")},
			wantErr: false,
		},
		{
			name: "The key does not exist",
			url:  path + "scores/kkk",
			want: struct {
				statusCode int
				body       []byte
			}{statusCode: http.StatusInternalServerError, body: []byte("kkk not exist\n")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := http.Get(tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer got.Body.Close()
			bytes, _ := io.ReadAll(got.Body)
			if got.StatusCode != tt.want.statusCode {
				t.Errorf("Get() got = %v, want %v", got.StatusCode, tt.want.statusCode)
			}
			if !reflect.DeepEqual(bytes, tt.want.body) {
				t.Errorf("Get() got = %s, want %s", bytes, tt.want.body)
			}
		})
	}
}
