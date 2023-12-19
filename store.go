package redisstore

import (
    "bytes"
    "encoding/gob"
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

func (s *RedisStore) Get(r *http.Request, name string) (*sessions.Session, error) {
    return sessions.GetRegistry(r).Get(s, name)
}

func (s *RedisStore) New(r *http.Request, name string) (*sessions.Session, error) {
    ss := sessions.NewSession(s, name)
    opts := s.Options
    ss.Options = opts
    ss.IsNew = true

    c, err := r.Cookie(name)
    if err != nil {
        return ss, nil
    }
    ss.ID = c.Value

    return ss, err
}

func (s *RedisStore) Save(r *http.Request, w http.ResponseWriter, ss *sessions.Session) error {

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

    // save
    var b bytes.Buffer
    enc := gob.NewEncoder(&b)
    if err := enc.Encode(ss); err != nil {
        return err
    }

    return s.Client.Set(r.Context(), ss.ID, b, time.Duration(ss.Options.MaxAge)*time.Second).Err()
}
