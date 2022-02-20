/*
 * MailPump - Copyright (C) 2022 Zane van Iperen.
 *    Contact: zane@zanevaniperen.com
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 2, and only
 * version 2 as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
 */

package imap

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-sasl"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func BuildTestIMAPServer(t *testing.T) (*server.Server, string, *memory.Mailbox) {
	be := memory.New()
	user, err := be.Login(nil, "username", "password")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	mb, err := user.GetMailbox("INBOX")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	mailbox := mb.(*memory.Mailbox)
	mailbox.Messages = nil

	s := server.New(be)
	t.Cleanup(func() { _ = s.Close() })

	s.AllowInsecureAuth = true

	l, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	go func() { err = s.Serve(l) }()

	return s, l.Addr().String(), mailbox
}

func TestNormal(t *testing.T) {
	_, address, _ := BuildTestIMAPServer(t)

	c, err := client.Dial(address)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Cleanup(func() { _ = c.Logout() })

	c.SetDebug(os.Stderr)

	a := NewNormalAuthenticator("username", "password")

	err = a.Authenticate(c)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}

func TestOAuthBearer(t *testing.T) {
	srv, address, _ := BuildTestIMAPServer(t)

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	now := time.Now()
	tf := func() time.Time { return now }
	jwt.TimeFunc = tf
	t.Cleanup(func() { jwt.TimeFunc = time.Now })

	tok := jwt.New(jwt.SigningMethodES256)
	tok.Header["kid"] = "mailpump"
	tok.Claims = jwt.RegisteredClaims{
		Issuer:    "mailpump test",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
		NotBefore: jwt.NewNumericDate(now),
		Subject:   "username",
	}

	signedTok, err := tok.SignedString(privKey)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	srv.EnableAuth("OAUTHBEARER", func(conn server.Conn) sasl.Server {
		return sasl.NewOAuthBearerServer(func(opts sasl.OAuthBearerOptions) *sasl.OAuthBearerError {
			clientTok, err := jwt.Parse(opts.Token, func(token *jwt.Token) (interface{}, error) {
				if token.Header["kid"] != tok.Header["kid"] {
					return nil, errors.New("invalid kid")
				}

				return &privKey.PublicKey, nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodES256.Name}))

			if !clientTok.Valid {
				return &sasl.OAuthBearerError{Status: "forbidden", Schemes: "bearer"}
			}

			if err := clientTok.Claims.Valid(); err != nil {
				t.Error(err)
				return &sasl.OAuthBearerError{Status: "forbidden", Schemes: "bearer"}
			}

			if clientTok.Claims.(jwt.MapClaims)["sub"] != "username" {
				return &sasl.OAuthBearerError{Status: "forbidden", Schemes: "bearer"}
			}

			if err != nil {
				t.Error(err)
				return &sasl.OAuthBearerError{Status: "invalid_request", Schemes: "bearer"}
			}

			return nil
		})
	})

	c, err := client.Dial(address)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Cleanup(func() { _ = c.Logout() })

	c.SetDebug(os.Stderr)

	a := NewOAuthBearerAuthenticator("username", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: signedTok}))

	err = a.Authenticate(c)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}
