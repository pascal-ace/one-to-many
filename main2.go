package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"io"
	"net/http"
	"time"
)

const (
	rtcpPLIInterval = time.Second * 3
)

var (
	broadcasterVideoTrack *webrtc.TrackLocalStaticRTP
	haveBroadcaster = false
	printSDP = false
)


func createSubscribe(w http.ResponseWriter, r *http.Request) {
	sdp := webrtc.SessionDescription{}
	if err := json.NewDecoder(r.Body).Decode(&sdp); err != nil {
		panic(err)
	}
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
	if printSDP {
		fmt.Printf("[SUB] OFFER FROM BROWSER #########\n%v\n", peerConnection.RemoteDescription().SDP)
	}
	_, err = peerConnection.AddTrack(broadcasterVideoTrack)
	if err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("[SUB] Connection State has changed %s \n", connectionState.String())
	})
/*	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()*/

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete
	/*if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}
*/
	output, err := json.MarshalIndent(peerConnection.LocalDescription(), "", "  ")
	if err != nil {
		panic(err)
	}
	if printSDP {
		fmt.Printf("[SUB] ANSWER TO BROWSER #########\n%v\n", string(output))
	}
	fmt.Println(broadcasterVideoTrack)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(output); err != nil {
		panic(err)
	}
}

func createBroadcast(w http.ResponseWriter, r *http.Request) {
	if haveBroadcaster {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("already have a broadcaster"))
		return
	}
	haveBroadcaster = true
	fmt.Println("hit!")
	sdp := webrtc.SessionDescription{}

	if err := json.NewDecoder(r.Body).Decode(&sdp); err != nil {
		panic(err)
	}

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
		fmt.Printf("[BRO] Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it
		go func() {
			ticker := time.NewTicker(rtcpPLIInterval)
			for range ticker.C {
				if rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(remoteTrack.SSRC())}}); rtcpSendErr != nil {
					fmt.Println(rtcpSendErr)
				}
			}
		}()

		// Create a local track, all our SFU clients will be fed via this track
		broadcasterVideoTrack, err = webrtc.NewTrackLocalStaticRTP(remoteTrack.Codec().RTPCodecCapability, "video", "pion")
		if err != nil {
			panic(err)
		}
		//localTrackChan <- localTrack

		rtpBuf := make([]byte, 1400)
		for {
			i, _, readErr := remoteTrack.Read(rtpBuf)
			if readErr != nil {
				panic(readErr)
			}

			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err = broadcasterVideoTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				panic(err)
			}
		}
	})

	if err = peerConnection.SetRemoteDescription(sdp); err != nil {
		panic(err)
	}
	if printSDP {
		fmt.Printf("[BRO] OFFER FROM BROWSER #########\n%v\n", peerConnection.RemoteDescription().SDP)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	output, err := json.MarshalIndent(peerConnection.LocalDescription(), "", "  ")
	if err != nil {
		panic(err)
	}
	if printSDP {
		fmt.Printf("[BRO] ANSWER TO BROWSER #########\n%v\n", string(output))
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(output); err != nil {
		panic(err)
	}
}

func main() {
	// trackLocals = map[string]*webrtc.TrackLocalStaticRTP{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	http.HandleFunc("/broadcast", createBroadcast)
	http.HandleFunc("/subscribe", createSubscribe)

	fmt.Println("Server has started on http://localhost:8000")
	panic(http.ListenAndServe(":8000", nil))
}
