package sdk

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type args struct {
	Port  string
	UUID  string
	Event string
	Info  string
}

type Info struct {
	Application struct {
		Font            string `json:"font"`
		Language        string `json:"language"`
		Platform        string `json:"platform"`
		PlatformVersion string `json:"platformVersion"`
		Version         string `json:"version"`
	} `json:"application"`
	Plugin struct {
		UUID    string `json:"uuid"`
		Version string `json:"version"`
	} `json:"plugin"`
	DevicePixelRatio int `json:"devicePixelRatio"`
	Colors           struct {
		ButtonPressedBackgroundColor string `json:"buttonPressedBackgroundColor"`
		ButtonPressedBorderColor     string `json:"buttonPressedBorderColor"`
		ButtonPressedTextColor       string `json:"buttonPressedTextColor"`
		DisabledColor                string `json:"disabledColor"`
		HighlightColor               string `json:"highlightColor"`
		MouseDownColor               string `json:"mouseDownColor"`
	} `json:"colors"`
	Devices []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Size struct {
			Columns int `json:"columns"`
			Rows    int `json:"rows"`
		} `json:"size"`
		Type int `json:"type"`
	} `json:"devices"`
}

type HandlerFunc func(*Plugin, interface{})

type Plugin struct {
	uuid string
	info *Info

	conn *websocket.Conn
	sync.Mutex

	handlers map[string]HandlerFunc
}

func NewPlugin() (*Plugin, error) {
	args := &args{}

	flag.StringVar(&args.Port, "port", "", "")
	flag.StringVar(&args.UUID, "pluginUUID", "", "")
	flag.StringVar(&args.Event, "registerEvent", "", "")
	flag.StringVar(&args.Info, "info", "", "")

	flag.Parse()

	if len(os.Args) != 9 {
		log.Fatalf("expected 9 arguments, got %d", len(os.Args))
	}

	info := &Info{}

	if err := json.Unmarshal([]byte(args.Info), info); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws://localhost:"+args.Port, nil)
	if err != nil {
		return nil, err
	}

	r := struct {
		Event string `json:"event"`
		UUID  string `json:"uuid"`
	}{
		Event: args.Event,
		UUID:  args.UUID,
	}

	if err := wsjson.Write(ctx, conn, r); err != nil {
		return nil, err
	}

	return &Plugin{uuid: args.UUID, info: info, conn: conn, handlers: make(map[string]HandlerFunc)}, nil
}

func (p *Plugin) Handle(path string, f HandlerFunc) {
	p.Lock()
	defer p.Unlock()

	p.handlers[path] = f

	log.Printf("registered %s", path)
}

func (p *Plugin) Run() error {
	for {
		ctx := context.Background()

		_, r, err := p.conn.Reader(ctx)
		if err != nil {
			log.Println(err)
		}

		b, err := ioutil.ReadAll(r)
		if err != nil {
			log.Println(err)
		}

		envelope := &EventEnvelope{}

		err = json.Unmarshal(b, envelope)
		if err != nil {
			log.Println(err)
		}

		if f, ok := p.handlers[envelope.Path]; ok {
			log.Printf("executing handler for %q", envelope.Path)
			f(p, envelope.Event)
		} else {
			log.Printf("no handler found for %q", envelope.Path)
		}
	}
}
