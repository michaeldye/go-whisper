package whisper

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pborman/uuid"
)

type Method string

const (
	CHECK_IDENTITY Method = "shh_hasIdentity"
	NEW_IDENTITY   Method = "shh_newIdentity"
	POST           Method = "shh_post"
	NEW_FILTER     Method = "shh_newFilter"
	GET_MSGS       Method = "shh_getMessages"
)

type WhisperRPCMsg struct {
	JsonRPC string      `json:"jsonrpc"`
	Id      string      `json:"id"`
	Result  interface{} `json:"result,omitempty"` // for generic handling of Result data
}

func (w WhisperRPCMsg) String() string {
	return fmt.Sprintf("JsonRPC: %v, Id: %v, Result: %v", w.JsonRPC, w.Id, w.Result)
}

type WhisperRPCOutgoingMsg struct {
	*WhisperRPCMsg
	Method Method        `json:"method"`
	Params []interface{} `json:"params"`
}

func (w WhisperRPCOutgoingMsg) String() string {
	return fmt.Sprintf("WhisperRPCMsg: %v, Method: %v, Params: %v", w.WhisperRPCMsg, w.Method, w.Params)
}

type Result struct {
	Hash    string `json:"hash"`
	Ttl     int    `json:"ttl"`
	Sent    int64  `json:"sent"`
	From    string `json:"from"`
	To      string `json:"to"`
	Payload string `json:"payload"` // probably a serialized message but that's provider-specific; a string for debugging convenience
}

func (r Result) String() string {
	return fmt.Sprintf("Hash: %s, Ttl: %d, Sent: %d, From: %s, To: %s, Payload: %s", r.Hash, r.Ttl, r.Sent, r.From, r.To, r.Payload)
}

func (r *Result) UnmarshalJSON(data []byte) error {
	if data == nil {
		return errors.New("Result type cannot unmarshal data: nil")
	} else {

		type TResult Result
		var t *TResult

		if err := json.Unmarshal(data, &t); err != nil {
			return err
		} else if bPayload, err := hex.DecodeString(t.Payload[2:]); err != nil {
			return err
		} else {
			// set fields on real type
			r.Hash = t.Hash
			r.Ttl = t.Ttl
			r.Sent = t.Sent
			r.From = t.From
			r.To = t.To
			r.Payload = string(bPayload[:])
			return nil
		}
	}
}

type WhisperRPCIncomingMsg interface{}

// need multiple concrete implementations b/c whisper isn't consistent with types in result field
type WhisperRPCIncomingMsgSingleStr struct {
	*WhisperRPCMsg
	Result string `json:"result"`
}

type WhisperRPCIncomingMsgSingleBool struct {
	*WhisperRPCMsg
	Result bool `json:"result"`
}

type WhisperRPCIncomingMsgMulti struct {
	*WhisperRPCMsg
	Result []Result `json:"result"`
}

func NewWhisperRPCOutgoingMsg(method Method, params []interface{}) *WhisperRPCOutgoingMsg {
	return &WhisperRPCOutgoingMsg{
		WhisperRPCMsg: &WhisperRPCMsg{
			JsonRPC: "2.0",
			Id:      uuid.New(),
		},
		Method: method,
		Params: params,
	}
}

func WrapParam(param interface{}) []interface{} {
	wrapped := make([]interface{}, 0)
	wrapped = append(wrapped, param)
	return wrapped
}

type transformVal func(val interface{}) (interface{}, error)

func transNoop(val interface{}) (interface{}, error) {
	return val, nil
}

// conforms to transformVal signature; TODO: move to custom marshaling
func toHex(val interface{}) (interface{}, error) {
	switch val.(type) {
	case int:
		return fmt.Sprintf("%#x", val), nil
	case string:
		return fmt.Sprintf("%#x", []byte(val.(string))), nil
	case [][]string: // TODO: support nested ints perhaps later
		vals := val.([][]string)

		hexed := make([][]string, len(vals))

		for ix, vi := range vals {
			hexed[ix] = make([]string, len(vals[ix]))
			for jx, vj := range vi {
				if hexVj, err := toHex(vj); err != nil {
					return hexed, err
				} else {
					hexed[ix][jx] = hexVj.(string)
				}
			}
		}
		return hexed, nil
	default:
		return "", fmt.Errorf("Unable to convert arg of type %T to hex: %v", val, val)
	}
}

func SimpleHexParam(param string) ([]interface{}, error) {

	var myVal interface{}

	if transformed, err := SingleMsgParam(param, toHex); err != nil {
		return nil, err
	} else {
		myVal = transformed
	}

	return WrapParam(myVal), nil
}

func SingleMsgParam(param interface{}, transFn transformVal) (interface{}, error) {

	switch param.(type) {
	case []string:
		enclosing := make([]interface{}, 0)

		for _, param := range param.([]string) {
			if trans, err := transFn(param); err != nil {
				return nil, err
			} else {
				enclosing = append(enclosing, trans)
			}
		}
		return enclosing, nil
	default:
		// let transform handle the types if they are not collections
		if trans, err := transFn(param); err != nil {
			return nil, err
		} else {
			return trans, nil
		}
	}
}

func MapMsgParams(params map[string]interface{}, transFn transformVal) ([]interface{}, error) {
	encoded := make(map[string]interface{})

	for k, v := range params {
		if trans, err := SingleMsgParam(v, transFn); err != nil {
			return nil, err
		} else {
			encoded[k] = trans
		}
	}

	return WrapParam(encoded), nil
}

func TopicMsgParams(id string, toId string, topics []string, payload string, ttl int, priority int) ([]interface{}, error) {

	// create encoded params first ...
	if hTopics, err := SingleMsgParam(topics, toHex); err != nil {
		return nil, err
	} else if hPayload, err := SingleMsgParam(payload, toHex); err != nil {
		return nil, err
	} else if hTtl, err := SingleMsgParam(ttl, toHex); err != nil {
		return nil, err
	} else if hPriority, err := SingleMsgParam(priority, toHex); err != nil {
		return nil, err
	} else {
		params := make(map[string]interface{})
		params["topics"] = hTopics
		params["payload"] = hPayload
		params["ttl"] = hTtl
		params["priority"] = hPriority

		// .. then bundle in the already-encoded ones
		params["from"] = id
		params["to"] = toId

		return MapMsgParams(params, transNoop)
	}
}

type ResultFilter func(r Result) bool

func filterIf(results []Result, filterFn ...ResultFilter) []Result {
	retained := make([]Result, 0)

	// O(n^2) be careful here
outer:
	for _, r := range results {
		for _, fn := range filterFn {
			if fn(r) {
				glog.V(5).Infof("Excluding record w/ hash %v b/c it failed filter function %v", r.Hash, fn)
				continue outer
			}
		}

		glog.V(5).Infof("Retaining record w/ hash %v", r.Hash)
		retained = append(retained, r)
	}

	return retained
}

func newFilter(client *http.Client, url string, topics interface{}) (string, error) {
	paramMap := make(map[string]interface{}, 1)
	paramMap["topics"] = topics

	if msg, err := MapMsgParams(paramMap, toHex); err != nil {
		return "", err
	} else if returned, err := WhisperSend(client, url, NEW_FILTER, msg, 5); err != nil {
		return "", err
	} else if reflect.TypeOf(returned) != reflect.TypeOf(WhisperRPCIncomingMsgSingleStr{}) {
		return "", fmt.Errorf("Unexpected msg type: %T", returned)
	} else {
		glog.V(4).Infof("Created new whisper filter: %v", msg)

		filterId := returned.(WhisperRPCIncomingMsgSingleStr).Result
		glog.V(4).Infof("FilterId: %s", filterId)

		return filterId, nil
	}
}

// WhisperReader returns a function that is called to poll for incoming messages on given topics for a specific duration
func WhisperReader(url string, topics interface{}) func(time.Duration, int64) ([]Result, error) {

	client := newClient()

	hashes := make(map[string]int64, 0)

	addHash := func(hash string, sent int64, ttl int) {
		hashes[hash] = sent + int64(ttl*2) // we don't mind holding on to hashes for longer than their TTL to be sure
		glog.V(4).Infof("Added hash %v to known list", hash)
	}

	// filter out of return if lambdas return true for given Result
	filterKnownHashFn := func(r Result) bool {
		// inefficient; need to ensure hashes stays small
		for k, _ := range hashes {
			if k == r.Hash {
				glog.V(5).Infof("Message w/ hash %v filtered b/c it matches a known hash", k)
				return true
			}
		}

		// add this one to known hashes to ensure duplicates in this set are handled
		addHash(r.Hash, r.Sent, r.Ttl)
		return false
	}

	// the duration is the duration b/n poll checks; a -1 readTimeoutS signifies no timeout
	return func(duration time.Duration, readTimeoutS int64) ([]Result, error) {

		purge := func(results []Result) []Result {
			// purge expired hashes
			for hash, expiration := range hashes {
				if time.Now().Unix() > expiration {
					delete(hashes, hash) // wow, safe in Go!
				}
			}

			glog.V(5).Infof("Pre-filtered whisper data: %v", justHashes(results))
			// do filtering
			filtered := filterIf(results, filterKnownHashFn)
			glog.V(5).Infof("Post-filtered whisper data: %v", justHashes(filtered))

			return filtered
		}

		// blocks until it gets results or timeout
		results, err := read(client, url, topics, duration, readTimeoutS, purge)
		if err != nil {
			return []Result{}, err
		}

		return results, nil
	}
}

func justHashes(results []Result) []string {
	acc := make([]string, 0)

	for _, r := range results {
		acc = append(acc, r.Hash)
	}

	return acc
}

func read(client *http.Client, url string, topics interface{}, duration time.Duration, readTimeoutS int64, filter func([]Result) []Result) ([]Result, error) {
	// hex-encoded value that whisper provides
	filterId := ""

	glog.V(2).Infof("Polling for incoming whisper messages at interval %s. Read timeout set to: %d", duration, readTimeoutS)

	start := time.Now().Unix()

	for {
		glog.V(5).Infof("Polling for messages at interval %v. Timeout occurs at: %v", duration, start+readTimeoutS)

		// could be -1 in which case it will block and poll forever, which is a supported use case
		if readTimeoutS > -1 && time.Now().Unix() > start+readTimeoutS {
			glog.Infof("Read timeout exceeded, ending whisper poll loop")
			return make([]Result, 0), nil
		} else {
			if filterId == "" {
				// set up filter b/c it's not set
				fid, err := newFilter(client, url, topics)
				if err != nil {
					return nil, err
				}

				filterId = fid
			}

			if returned, err := WhisperSend(client, url, GET_MSGS, WrapParam(filterId), 5); err != nil {
				return nil, err
			} else if returned == nil {
				// filter timeout, recreate it
				filterId = ""
			} else if reflect.TypeOf(returned) != reflect.TypeOf(WhisperRPCIncomingMsgMulti{}) {
				return nil, fmt.Errorf("Unexpected msg type: %T. Content: %v", returned, returned)
			} else {
				filtered := filter(returned.(WhisperRPCIncomingMsgMulti).Result)
				if len(filtered) > 0 {
					return filtered, nil
				}
			}
		}

		glog.V(5).Infof("Yielding and sleeping for specified poll interval %v", duration)
		time.Sleep(duration)
	}
}

func newClient() *http.Client {
	return &http.Client{
		Timeout: time.Minute * 3, // make this really long in case system is hosed
	}
}

func WhisperSend(client *http.Client, url string, method Method, params []interface{}, logV glog.Level) (WhisperRPCIncomingMsg, error) {
	if client == nil {
		client = newClient()
	}

	out := NewWhisperRPCOutgoingMsg(method, params)

	serial, err := json.Marshal(out)

	if err != nil {
		return nil, err
	}
	glog.V(logV).Infof("Sending: %v", string(serial[:]))
	response, err := client.Post(url, "application/json", strings.NewReader(string(serial[:])))

	if err != nil {
		return nil, fmt.Errorf("Unable to make RPC calls of ethereum instance: Error: %v. Original request: %v", err, string(serial[:]))
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("RPC call to ethereum instance returned non-OK status. Response: %v", response)
	}

	if content, err := ioutil.ReadAll(response.Body); err != nil {
		return nil, err
	} else {
		var failsafe WhisperRPCMsg

		var multi WhisperRPCIncomingMsgMulti
		var sStr WhisperRPCIncomingMsgSingleStr
		var sBool WhisperRPCIncomingMsgSingleBool

		// TODO: add check of Id in returned message against Id in sent message
		if mErr := json.Unmarshal(content, &multi); mErr == nil {
			return multi, nil
		} else if bErr := json.Unmarshal(content, &sBool); bErr == nil {
			return sBool, nil
		} else if sErr := json.Unmarshal(content, &sStr); sErr == nil {
			return sStr, nil
		} else if err := json.Unmarshal(content, &failsafe); err == nil {

			return failsafe, nil
		} else {
			return nil, fmt.Errorf("Error deserializing response from whisper client as a WhisperRPCIncomingMsg type. Returned content: %s", string(content))
		}
	}
}
