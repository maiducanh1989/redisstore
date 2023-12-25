package redisstore

import (
    "bytes"
    "encoding/gob"
    "net/http"
    "strings"
    "time"

    "github.com/gorilla/sessions"
    "github.com/oklog/ulid/v2"
    "github.com/redis/go-redis/v9"
)

type (
    RedisStore struct {
        sessions.Store
        Options *sessions.Options
        Client  *redis.Client
    }
)

func (store *RedisStore) Get(request *http.Request, name string) (*sessions.Session, error) {

    sess, _ := store.New(request, name)

    c, err := request.Cookie(name)
    if err != nil {
        return sess, nil
    }

    sess.ID = c.Value
    sess.IsNew = false

    b, err := store.Client.Get(request.Context(), c.Value).Bytes()
    if err != nil {
        return sess, nil
    }

    bf := bytes.NewBuffer(b)
    dec := gob.NewDecoder(bf)
    if err := dec.Decode(&sess.Values); err != nil {
    }

    return sess, nil

}

func (store *RedisStore) New(_ *http.Request, name string) (*sessions.Session, error) {

    sess := sessions.NewSession(store, name)
    sess.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   86400 * 30, // 30 days: 86400 * 30
        Secure:   true,
        HttpOnly: true,
    }
    if store.Options != nil {
        sess.Options = store.Options
    }
    sess.IsNew = true

    return sess, nil

}

func (store *RedisStore) Save(request *http.Request, writer http.ResponseWriter, sess *sessions.Session) error {

    // Delete if max-age is <= 0

    if sess.Options.MaxAge <= 0 {
        if err := store.Client.Del(request.Context(), sess.ID).Err(); err != nil {
            return err
        }
        sess.ID = ""
    }

    // new cookie

    if sess.ID == "" {
        sess.ID = strings.ToLower(ulid.Make().String())
        http.SetCookie(writer, sessions.NewCookie(sess.Name(), sess.ID, sess.Options))
    }

    // save session values

    var bf bytes.Buffer
    enc := gob.NewEncoder(&bf)
    if err := enc.Encode(sess.Values); err != nil {
        return err
    }

    return store.Client.Set(
        request.Context(),
        sess.ID,
        bf.Bytes(),
        time.Duration(sess.Options.MaxAge)*time.Second,
    ).Err()
}
