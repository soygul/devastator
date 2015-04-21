package main

import (
	"crypto/tls"
	"crypto/x509"
	"strconv"
	"testing"
)

func TestLen(t *testing.T) {
	a, _ := strconv.Atoi("12344324")
	t.Log(a)
}

func TestListener(t *testing.T) {
	msg1 := "Lorem ipsum dolor sit amet, consectetur adipiscing elit."
	msg2 := "In sit amet lectus felis, at pellentesque turpis."
	msg3 := "Nunc urna enim, cursus varius aliquet ac, imperdiet eget tellus."
	msg4 := randString(45000)
	msg5 := randString(500000)

	host := "localhost:" + Conf.App.Port
	cert, privKey, _ := genCert("localhost", 0, nil, nil, 512, "localhost", "devastator")
	listener, err := Listen(cert, privKey, host, Conf.App.Debug)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	go listener.Accept(func(conn *Conn, session *Session, msg []byte) {
		certs := conn.ConnectionState().PeerCertificates
		if len(certs) > 0 {
			t.Logf("Client connected with client certificate subject: %v\n", certs[0].Subject)
		}
	}, func(conn *Conn, session *Session) {
	})

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(cert)
	if !ok {
		panic("failed to parse root certificate")
	}

	tlsConf := &tls.Config{RootCAs: roots}
	conn, err := tls.Dial("tcp", host, tlsConf)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	newconn := NewConn(conn, 0, 0)

	send(t, newconn, "4\nping")
	send(t, newconn, "56\n"+msg1)
	send(t, newconn, "56\n"+msg1)
	send(t, newconn, "49\n"+msg2)
	send(t, newconn, "64\n"+msg3)
	send(t, newconn, "45000\n"+msg4)
	send(t, newconn, "56\n"+msg1)
	send(t, newconn, "500000\n"+msg5)
	send(t, newconn, "56\n"+msg1)
	send(t, newconn, "5\nclose")

	// t.Logf("\nconn:\n%+v\n\n", conn)
	// t.Logf("\nconn.ConnectionState():\n%+v\n\n", conn.ConnectionState())
	// t.Logf("\ntls.Config:\n%+v\n\n", tlsConf)
}

func send(t *testing.T, conn *Conn, msg string) {
	n, err := conn.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Error while writing message to connection %v", err)
	}
	t.Logf("Sending message to listener from client: %v (%v bytes)", msg, n)
}
