// SFTP server for users with Flounder accounts
// 	A lot of this is copied from SFTPGo, but simplified for our use case.
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Connection struct {
	User string
}

func (con *Connection) Fileread(request *sftp.Request) (io.ReaderAt, error) {
	// check user perms -- cant read others hidden files
	fullpath := path.Join(c.FilesDirectory, filepath.Clean(request.Filepath))
	f, err := os.OpenFile(fullpath, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (con *Connection) Filewrite(request *sftp.Request) (io.WriterAt, error) {
	// check user perms -- cant write others files
	fullpath := path.Join(c.FilesDirectory, filepath.Clean(request.Filepath))
	userDir := getUserDirectory(con.User) // NOTE -- not cross platform
	if strings.HasPrefix(fullpath, userDir) {
		f, err := os.OpenFile(fullpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return nil, err
		}
		return f, nil
	} else {
		return nil, fmt.Errorf("Invalid permissions")
	}
}

func (conn *Connection) Filelist(request *sftp.Request) (sftp.ListerAt, error) {
	fullpath := path.Join(c.FilesDirectory, filepath.Clean(request.Filepath))
	if strings.Contains(request.Filepath, ".hidden") {
		return nil, fmt.Errorf("Invalid permissions") // TODO fix better
	}
	switch request.Method {
	case "List":
		f, err := os.Open(fullpath)
		if err != nil {
			return nil, err
		}
		fileInfo, err := f.Readdir(-1)
		if err != nil {
			return nil, err
		}
		return listerat(fileInfo), nil
	case "Stat":
		stat, err := os.Stat(fullpath)
		if err != nil {
			return nil, err
		}
		return listerat([]os.FileInfo{stat}), nil
	}
	return nil, fmt.Errorf("Invalid command")
}

func (conn *Connection) Filecmd(request *sftp.Request) error {
	// remove, rename, setstat? find out
	fullpath := path.Join(c.FilesDirectory, filepath.Clean(request.Filepath))
	userDir := getUserDirectory(conn.User) // NOTE -- not cross platform
	writePerms := strings.HasPrefix(fullpath, userDir)
	var err error
	if writePerms {
		switch request.Method {
		case "Remove":
			err = os.Remove(fullpath)
		case "Mkdir":
			err = os.Mkdir(fullpath, 0755)
		}
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unauthorized")
	}
	// Rename, Mkdir
	return nil
}

// TODO hide hidden folders
// Users have write persm on their files, read perms on all

func buildHandlers(connection *Connection) sftp.Handlers {
	return sftp.Handlers{
		connection,
		connection,
		connection,
		connection,
	}
}

// Based on example server code from golang.org/x/crypto/ssh and server_standalone
func runSFTPServer() {
	if !c.EnableSFTP {
		return
	}
	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in
			// a production setting.
			if isOkUsername(c.User()) != nil { // extra check, probably unnecessary
				return nil, fmt.Errorf("Invalid username")
			}
			_, _, err := checkLogin(c.User(), string(pass))
			// TODO maybe give admin extra permissions?
			if err != nil {
				return nil, fmt.Errorf("password rejected for %q", c.User())
			} else {
				log.Printf("Login: %s\n", c.User())
				return nil, nil
			}
		},
	}

	// TODO generate key automatically
	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key", err)
	}

	config.AddHostKey(private)

	listener, err := net.Listen("tcp", "0.0.0.0:2024")
	if err != nil {
		log.Fatal("failed to listen for connection", err)
	}

	log.Printf("SFTP server listening on %v\n", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go acceptInboundConnection(conn, config)
	}
}

func acceptInboundConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("panic in AcceptInboundConnection: %#v stack strace: %v", r, string(debug.Stack()))
		}
	}()
	ipAddr := GetIPFromRemoteAddress(conn.RemoteAddr().String())
	log.Println("Request from IP " + ipAddr)
	limiter := getVisitor(ipAddr)
	if limiter.Allow() == false {
		conn.Close()
		return
	}
	// Before beginning a handshake must be performed on the incoming net.Conn
	// we'll set a Deadline for handshake to complete, the default is 2 minutes as OpenSSH
	conn.SetDeadline(time.Now().Add(2 * time.Minute))

	// Before use, a handshake must be performed on the incoming net.Conn.
	sconn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("failed to accept an incoming connection: %v", err)
		return
	}
	log.Println("login detected:", sconn.User())
	fmt.Fprintf(os.Stderr, "SSH server established\n")
	// handshake completed so remove the deadline, we'll use IdleTimeout configuration from now on
	conn.SetDeadline(time.Time{})

	defer conn.Close()

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	channelCounter := int64(0)
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of an SFTP session, this is "subsystem"
		// with a payload string of "<length=4>sftp"
		fmt.Fprintf(os.Stderr, "Incoming channel: %s\n", newChannel.ChannelType())
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			fmt.Fprintf(os.Stderr, "Unknown channel type: %s\n", newChannel.ChannelType())
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Println("could not accept channel.", err)
			continue
		}

		channelCounter++
		fmt.Fprintf(os.Stderr, "Channel accepted\n")

		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env".  Here we handle only the
		// "subsystem" request.
		go func(in <-chan *ssh.Request) {
			for req := range in {
				fmt.Fprintf(os.Stderr, "Request: %v\n", req.Type)
				ok := false
				switch req.Type {
				case "subsystem":
					fmt.Fprintf(os.Stderr, "Subsystem: %s\n", req.Payload[4:])
					if string(req.Payload[4:]) == "sftp" {
						ok = true
					}
				}
				fmt.Fprintf(os.Stderr, " - accepted: %v\n", ok)
				req.Reply(ok, nil)
			}
		}(requests)
		connection := Connection{sconn.User()}
		root := buildHandlers(&connection)
		server := sftp.NewRequestServer(channel, root)
		if err := server.Serve(); err == io.EOF {
			server.Close()
			log.Println("sftp client exited session.")
		} else if err != nil {
			log.Println("sftp server completed with error:", err)
			return
		}
	}
}

type listerat []os.FileInfo

// Modeled after strings.Reader's ReadAt() implementation
func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}
