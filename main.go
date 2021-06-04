package main

import (
	"encoding/json"
	"fmt"
	"github.com/pion/webrtc/v3"
	"net/http"
	"sync"
)

type pubStream struct {
	stream *webrtc.TrackLocal
}

var (
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
	listLock    sync.RWMutex
)

func createSubscribe(w http.ResponseWriter, r *http.Request) {
	sdp := webrtc.SessionDescription{}
	if err := json.NewDecoder(r.Body).Decode(&sdp); err != nil {
		panic(err)
	}

	// Create new PeerConnection
	// Everything below is the pion-WebRTC API! Thanks for using it ❤️.
	// Prepare the configuration
	//config := webrtc.Configuration{
	//	ICEServers: []webrtc.ICEServer{
	//		{
	//			URLs: []string{"stun:stun.l.google.com:19302"},
	//		},
	//	},
	//}
	//
	//// Create a MediaEngine object to configure the supported codec
	//m := &webrtc.MediaEngine{}
	//
	//// Setup the codecs you want to use.
	//// Only support VP8 and OPUS, this makes our WebM muxer code simpler
	//if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	//	RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	//	PayloadType:        96,
	//}, webrtc.RTPCodecTypeVideo); err != nil {
	//	panic(err)
	//}
	//if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	//	RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	//	PayloadType:        111,
	//}, webrtc.RTPCodecTypeAudio); err != nil {
	//	panic(err)
	//}
	//
	////audioBuilder = samplebuilder.New(10, &codecs.OpusPacket{}, 48000)
	////videoBuilder = samplebuilder.New(10, &codecs.VP8Packet{}, 90000)
	//
	//// Create the API object with the MediaEngine
	//api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	//
	//// Create a new RTCPeerConnection
	//peerConnection, err := api.NewPeerConnection(config)
	//if err != nil {
	//	panic(err)
	//}
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	if err = peerConnection.SetRemoteDescription(sdp); err != nil {
		panic(err)
	}

	if _, err = peerConnection.AddTrack(trackLocals["publisher"]); err != nil {
		panic(err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	output, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(trackLocals)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(output); err != nil {
		panic(err)
	}
}

func createBroadcast(w http.ResponseWriter, r *http.Request) {
	fmt.Println("hit!")
	sdp := webrtc.SessionDescription{}
	if err := json.NewDecoder(r.Body).Decode(&sdp); err != nil {
		panic(err)
	}
	// Create new PeerConnection
	// Everything below is the pion-WebRTC API! Thanks for using it ❤️.
	// Prepare the configuration
	//config := webrtc.Configuration{
	//	ICEServers: []webrtc.ICEServer{
	//		{
	//			URLs: []string{"stun:stun.l.google.com:19302"},
	//		},
	//	},
	//}
	//
	//// Create a MediaEngine object to configure the supported codec
	//m := &webrtc.MediaEngine{}
	//
	//// Setup the codecs you want to use.
	//// Only support VP8 and OPUS, this makes our WebM muxer code simpler
	//if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	//	RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	//	PayloadType:        96,
	//}, webrtc.RTPCodecTypeVideo); err != nil {
	//	panic(err)
	//}
	//if err := m.RegisterCodec(webrtc.RTPCodecParameters{
	//	RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
	//	PayloadType:        111,
	//}, webrtc.RTPCodecTypeAudio); err != nil {
	//	panic(err)
	//}
	//
	////audioBuilder = samplebuilder.New(10, &codecs.OpusPacket{}, 48000)
	////videoBuilder = samplebuilder.New(10, &codecs.VP8Packet{}, 90000)
	//
	//// Create the API object with the MediaEngine
	//api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	//
	//// Create a new RTCPeerConnection
	//peerConnection, err := api.NewPeerConnection(config)
	//if err != nil {
	//	panic(err)
	//}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		fmt.Println("ontrack")
		trackLocal := addTrack(t)
		buf := make([]byte, 1500)
		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}
			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
		//fmt.Println("ontrack stream", stream)
	})

	if err = peerConnection.SetRemoteDescription(sdp); err != nil {
		panic(err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	output, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(output); err != nil {
		panic(err)
	}
}

func addTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
	}()
	fmt.Println("handle E: ", t)
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	trackLocals["publisher"] = trackLocal
	return trackLocal
}

func main() {
	trackLocals = map[string]*webrtc.TrackLocalStaticRTP{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	http.HandleFunc("/broadcast", createBroadcast)
	http.HandleFunc("/subscribe", createSubscribe)

	fmt.Println("Server has started on :8000")
	panic(http.ListenAndServe(":8000", nil))
}
