// An example SFTP server implementation using the golang SSH package.
// Serves the whole filesystem visible to the user, and has a hard-coded username and password,
// so not for real use!
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
	"strings"

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

func (c *Connection) Filecmd(request *sftp.Request) error {
	// remove, rename, setstat? find out
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
			fmt.Fprintf(os.Stderr, "Login: %s\n", c.User())
			if err != nil {
				return nil, fmt.Errorf("password rejected for %q", c.User())
			} else {
				return nil, nil
			}
		},
	}

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key", err)
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2024")
	if err != nil {
		log.Fatal("failed to listen for connection", err)
	}
	fmt.Printf("Listening on %v\n", listener.Addr())

	nConn, err := listener.Accept()
	if err != nil {
		log.Fatal("failed to accept incoming connection", err)
	}

	// Before use, a handshake must be performed on the incoming net.Conn.
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Fatal("failed to handshake", err)
	}
	log.Println("login detected:", sconn.User())
	fmt.Fprintf(os.Stderr, "SSH server established\n")

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
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
			log.Fatal("could not accept channel.", err)
		}
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
		connection := Connection{"alex"}
		root := buildHandlers(&connection)
		server := sftp.NewRequestServer(channel, root)
		if err := server.Serve(); err == io.EOF {
			server.Close()
			log.Print("sftp client exited session.")
		} else if err != nil {
			log.Fatal("sftp server completed with error:", err)
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
