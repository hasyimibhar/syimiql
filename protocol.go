package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	protocolVersion3_0 = 196608
	sslRequestCode     = 80877103
)

type PgServer struct {
	listener net.Listener
}

func NewPgServer(addr string) (*PgServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &PgServer{listener: listener}, nil
}

func (s *PgServer) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *PgServer) Close() error {
	return s.listener.Close()
}

func (s *PgServer) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(conn)
	}
}

func (s *PgServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	if err := s.handleStartup(conn); err != nil {
		s.sendError(conn, err.Error())
		return
	}

	for {
		msgType, payload, err := s.readMessage(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}

		switch msgType {
		case 'Q':
			s.handleQuery(conn, payload)
		case 'X':
			return
		default:
			s.sendError(conn, fmt.Sprintf("unsupported message type: %c", msgType))
			return
		}
	}
}

func (s *PgServer) handleStartup(conn net.Conn) error {
	for {
		lengthBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, lengthBuf); err != nil {
			return err
		}
		length := int(binary.BigEndian.Uint32(lengthBuf))

		payload := make([]byte, length-4)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return err
		}

		version := binary.BigEndian.Uint32(payload[:4])

		if version == sslRequestCode {
			if _, err := conn.Write([]byte{'N'}); err != nil {
				return err
			}
			continue
		}

		if version != protocolVersion3_0 {
			return fmt.Errorf("unsupported protocol version: %d", version)
		}

		params := s.parseStartupParams(payload[4:])
		_ = params
		break
	}

	if err := s.sendAuthenticationOk(conn); err != nil {
		return err
	}

	if err := s.sendParameterStatus(conn, "server_version", "15.0"); err != nil {
		return err
	}
	if err := s.sendParameterStatus(conn, "client_encoding", "UTF8"); err != nil {
		return err
	}
	if err := s.sendParameterStatus(conn, "server_encoding", "UTF8"); err != nil {
		return err
	}

	if err := s.sendBackendKeyData(conn, 1234, 5678); err != nil {
		return err
	}

	if err := s.sendReadyForQuery(conn, 'I'); err != nil {
		return err
	}

	return nil
}

func (s *PgServer) parseStartupParams(data []byte) map[string]string {
	params := make(map[string]string)
	i := 0
	for i < len(data) {
		keyEnd := i
		for keyEnd < len(data) && data[keyEnd] != 0 {
			keyEnd++
		}
		if keyEnd >= len(data) {
			break
		}
		key := string(data[i:keyEnd])
		if key == "" {
			break
		}
		i = keyEnd + 1

		valEnd := i
		for valEnd < len(data) && data[valEnd] != 0 {
			valEnd++
		}
		value := string(data[i:valEnd])
		i = valEnd + 1

		params[key] = value
	}
	return params
}

func (s *PgServer) readMessage(conn net.Conn) (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	msgType := header[0]
	length := int(binary.BigEndian.Uint32(header[1:5]))

	payload := make([]byte, length-4)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, nil, err
	}

	return msgType, payload, nil
}

func (s *PgServer) handleQuery(conn net.Conn, payload []byte) {
	query := string(payload[:len(payload)-1])
	_ = query

	// TODO: implement this

	s.sendCommandComplete(conn, "OK")
	s.sendReadyForQuery(conn, 'I')
}

func (s *PgServer) sendAuthenticationOk(conn net.Conn) error {
	msg := make([]byte, 9)
	msg[0] = 'R'
	binary.BigEndian.PutUint32(msg[1:5], 8)
	binary.BigEndian.PutUint32(msg[5:9], 0)
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendParameterStatus(conn net.Conn, name, value string) error {
	payload := append([]byte(name), 0)
	payload = append(payload, []byte(value)...)
	payload = append(payload, 0)

	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'S'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendBackendKeyData(conn net.Conn, pid, secretKey int32) error {
	msg := make([]byte, 13)
	msg[0] = 'K'
	binary.BigEndian.PutUint32(msg[1:5], 12)
	binary.BigEndian.PutUint32(msg[5:9], uint32(pid))
	binary.BigEndian.PutUint32(msg[9:13], uint32(secretKey))
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendReadyForQuery(conn net.Conn, status byte) error {
	msg := []byte{'Z', 0, 0, 0, 5, status}
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendCommandComplete(conn net.Conn, tag string) error {
	payload := append([]byte(tag), 0)
	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'C'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}

func (s *PgServer) sendError(conn net.Conn, message string) error {
	payload := []byte{'S'}
	payload = append(payload, []byte("ERROR")...)
	payload = append(payload, 0)
	payload = append(payload, 'M')
	payload = append(payload, []byte(message)...)
	payload = append(payload, 0)
	payload = append(payload, 0)

	msg := make([]byte, 1+4+len(payload))
	msg[0] = 'E'
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	_, err := conn.Write(msg)
	return err
}
