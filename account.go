package whisper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
)

func AccountId(url string) (string, error) {
	// try reading from account file, if not set or invalid, will generate and overwrite

	// generate own
	gen := func() (string, error) {
		empty := make([]interface{}, 0)
		if id, err := WhisperSend(nil, url, NEW_IDENTITY, empty, 3); err != nil {
			return "", err
		} else {
			switch s := interface{}(id).(type) {
			case WhisperRPCIncomingMsgSingleStr:
				glog.V(3).Infof("Returned message from whisper generate id: %v", s.Result)
				if s.Result == "" {
					return "", errors.New("No returned whisper id from generate call, cannot proceed")
				}
				return s.Result, nil
			default:
				return "", fmt.Errorf("Unexpected return type: %T", id)
			}
		}
	}

	whisperFile := os.Getenv("CMTN_WHISPER_ADDRESS_PATH")

	var file *os.File
	var err error
	if _, err = os.Stat(whisperFile); os.IsNotExist(err) {
		file, err = os.Create(whisperFile)
		if err != nil {
			return "", err
		}
	} else {
		file, err = os.OpenFile(whisperFile, os.O_RDWR, 0600)
		if err != nil {
			return "", err
		}
	}

	defer file.Close()

	var id string

	if err != nil {
		return "", err
	} else {
		if data, err := ioutil.ReadFile(file.Name()); err != nil {
			return "", err
		} else {
			id = strings.Trim(string(data), "\n\r ")
			glog.V(2).Infof("Read whisper id from file: %v", id)

			// check the id
			if in, err := WhisperSend(nil, url, CHECK_IDENTITY, WrapParam(id), 5); err != nil {
				return "", err
			} else {
				switch s := interface{}(in).(type) {
				case WhisperRPCIncomingMsgSingleBool:
					glog.V(3).Infof("Returned message from whisper identity check: %v", s.Result)
					if !s.Result {
						glog.V(3).Infof("Client returned false for check identity call, generating own")

						newId, err := gen()
						if err != nil {
							return "", err
						}
						// attempt to write it

						_, err = file.Write([]byte(newId))
						if err != nil {
							return "", err
						} else {
							return newId, nil
						}
					}
					return id, nil
				default:
					return "", fmt.Errorf("Unexpected type: %T", in)
				}
			}
		}
	}
}
