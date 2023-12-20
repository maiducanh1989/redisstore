package redisstore

import (
    "bytes"
    "encoding/gob"
    "fmt"
    "net/http"
    "time"

    "github.com/gorilla/sessions"
    "github.com/oklog/ulid/v2"
    "github.com/redis/go-redis/v9"
)

type (
    RedisStore struct {
        sessions.Store

        Client  *redis.Client
        Options *sessions.Options
    }
)

// var ctx = context.Background()

func (s *RedisStore) New(r *http.Request, name string) (*sessions.Session, error) {

    fmt.Println("redisstore: new")
    
    ss := sessions.NewSession(s, name)
    ss.Options = s.Options
    ss.IsNew = true

    c, err := r.Cookie(name)

    if err != nil {
        return nil, err
    }
    fmt.Println("redisstore: new:", c)
    ss.ID = c.Value

    return ss, err

}

func (s *RedisStore) Save(r *http.Request, w http.ResponseWriter, ss *sessions.Session) error {

    fmt.Println("redisstore: save")

    // Delete if max-age is <= 0
    if ss.Options.MaxAge <= 0 {
        if err := s.Client.Del(r.Context(), ss.ID).Err(); err != nil {
            return err
        }
        http.SetCookie(w, sessions.NewCookie(ss.Name(), "", ss.Options))
        return nil
    }

    // generate session ID
    if ss.ID == "" {
        ss.ID = ulid.Make().String()
    }

    fmt.Println("redisstore: save:", ss.ID)

    // save
    var b bytes.Buffer
    enc := gob.NewEncoder(&b)
    if err := enc.Encode(ss); err != nil {
        return err
    }

    return s.Client.Set(r.Context(), ss.ID, b, time.Duration(ss.Options.MaxAge)*time.Second).Err()

}

func (s *RedisStore) Get(r *http.Request, name string) (*sessions.Session, error) {

    fmt.Println("redisstore: get")

    c, err := r.Cookie(name)
    if err != nil {
        return nil, err
    }

    b, err := s.Client.Get(r.Context(), c.Value).Bytes()
    if err != nil {
        return nil, err
    }

    buf := bytes.NewBuffer(b)
    dec := gob.NewDecoder(buf)
    ss := sessions.Session{}
    if err := dec.Decode(&ss); err != nil {
        return nil, err
    }

    return &ss, nil

}
