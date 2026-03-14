package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const streamSinkName = "musicplayerd_sink"

// StreamManager manages FFmpeg processes that capture audio from a dedicated
// PulseAudio null sink and stream it via RTP and/or HTTP.
//
// A virtual sink is created so that only mpv's audio is captured (not all
// system audio). A loopback module routes the virtual sink to the default
// output so local playback still works.
type StreamManager struct {
	cfg Config

	rtpCmd  *exec.Cmd
	httpCmd *exec.Cmd

	// PulseAudio module IDs for cleanup.
	sinkModuleID     string
	loopbackModuleID string
}

// NewStreamManager creates a stream manager from the daemon config.
func NewStreamManager(cfg Config) *StreamManager {
	return &StreamManager{cfg: cfg}
}

// SinkName returns the PulseAudio sink name that mpv should output to.
// Only valid after Start().
func (s *StreamManager) SinkName() string {
	return streamSinkName
}

// Start creates the virtual PulseAudio sink, loopback, and FFmpeg processes.
func (s *StreamManager) Start() error {
	if err := s.createVirtualSink(); err != nil {
		return fmt.Errorf("virtual sink: %w", err)
	}

	monitorSource := streamSinkName + ".monitor"

	if s.cfg.StreamRTPDest != "" {
		if err := s.startRTP(monitorSource); err != nil {
			return fmt.Errorf("rtp stream: %w", err)
		}
	}
	if s.cfg.StreamHTTPPort != "" {
		if err := s.startHTTP(monitorSource); err != nil {
			return fmt.Errorf("http stream: %w", err)
		}
	}
	return nil
}

// Stop kills all streaming processes and removes the virtual PulseAudio sink.
func (s *StreamManager) Stop() {
	if s.rtpCmd != nil && s.rtpCmd.Process != nil {
		log.Printf("stopping RTP stream")
		s.rtpCmd.Process.Kill()
		s.rtpCmd.Wait()
	}
	if s.httpCmd != nil && s.httpCmd.Process != nil {
		log.Printf("stopping HTTP stream")
		s.httpCmd.Process.Kill()
		s.httpCmd.Wait()
	}
	s.removeVirtualSink()
}

// createVirtualSink sets up a PulseAudio null sink and a loopback to the
// default output so mpv's audio is isolated for capture but still audible.
func (s *StreamManager) createVirtualSink() error {
	// Create a null sink for mpv to output to.
	out, err := exec.Command("pactl", "load-module", "module-null-sink",
		"sink_name="+streamSinkName,
		"sink_properties=device.description=MusicPlayerD",
	).Output()
	if err != nil {
		return fmt.Errorf("load module-null-sink: %w", err)
	}
	s.sinkModuleID = strings.TrimSpace(string(out))
	log.Printf("created virtual sink %s (module %s)", streamSinkName, s.sinkModuleID)

	// Loopback the virtual sink to the default output for local playback.
	out, err = exec.Command("pactl", "load-module", "module-loopback",
		"source="+streamSinkName+".monitor",
		"sink=@DEFAULT_SINK@",
		"latency_msec=50",
	).Output()
	if err != nil {
		// Non-fatal: streaming still works, just no local audio.
		log.Printf("warning: loopback module failed (no local audio): %v", err)
	} else {
		s.loopbackModuleID = strings.TrimSpace(string(out))
		log.Printf("loopback to default sink (module %s)", s.loopbackModuleID)
	}

	return nil
}

// removeVirtualSink unloads the PulseAudio modules created by createVirtualSink.
func (s *StreamManager) removeVirtualSink() {
	if s.loopbackModuleID != "" {
		exec.Command("pactl", "unload-module", s.loopbackModuleID).Run()
		log.Printf("removed loopback module %s", s.loopbackModuleID)
	}
	if s.sinkModuleID != "" {
		exec.Command("pactl", "unload-module", s.sinkModuleID).Run()
		log.Printf("removed virtual sink module %s", s.sinkModuleID)
	}
}

// startRTP launches an FFmpeg process that captures from the virtual sink monitor
// and outputs an RTP stream to the configured destination.
func (s *StreamManager) startRTP(source string) error {
	codec, muxer := s.codecArgs()

	args := []string{
		"-f", "pulse",
		"-i", source,
		"-acodec", codec,
		"-ab", s.cfg.StreamBitrate,
		"-f", "rtp",
	}
	if muxer != "" {
		args = append(args, "-muxer_options", muxer)
	}
	args = append(args, s.cfg.StreamRTPDest)

	s.rtpCmd = exec.Command("ffmpeg", args...)
	s.rtpCmd.Stdout = os.Stdout
	s.rtpCmd.Stderr = os.Stderr

	if err := s.rtpCmd.Start(); err != nil {
		return err
	}
	log.Printf("RTP stream started → %s (codec=%s bitrate=%s source=%s)", s.cfg.StreamRTPDest, codec, s.cfg.StreamBitrate, source)
	return nil
}

// startHTTP launches an FFmpeg process that serves an HTTP audio stream.
// FFmpeg's -listen 1 creates a simple single-client HTTP server.
func (s *StreamManager) startHTTP(source string) error {
	codec, _ := s.codecArgs()
	listenURL := fmt.Sprintf("http://0.0.0.0:%s/stream", s.cfg.StreamHTTPPort)
	contentType := s.contentType()

	args := []string{
		"-f", "pulse",
		"-i", source,
		"-acodec", codec,
		"-ab", s.cfg.StreamBitrate,
		"-f", s.outputFormat(),
		"-content_type", contentType,
		"-listen", "1",
		listenURL,
	}

	s.httpCmd = exec.Command("ffmpeg", args...)
	s.httpCmd.Stdout = os.Stdout
	s.httpCmd.Stderr = os.Stderr

	if err := s.httpCmd.Start(); err != nil {
		return err
	}
	log.Printf("HTTP stream started → %s (codec=%s bitrate=%s source=%s)", listenURL, codec, s.cfg.StreamBitrate, source)
	return nil
}

// codecArgs returns the FFmpeg codec name and optional muxer options for the configured format.
func (s *StreamManager) codecArgs() (codec string, muxerOpts string) {
	switch s.cfg.StreamFormat {
	case "opus":
		return "libopus", ""
	case "aac":
		return "aac", ""
	case "flac":
		return "flac", ""
	default: // mp3
		return "libmp3lame", ""
	}
}

// outputFormat returns the FFmpeg output format for HTTP streaming.
func (s *StreamManager) outputFormat() string {
	switch s.cfg.StreamFormat {
	case "opus":
		return "ogg"
	case "aac":
		return "adts"
	case "flac":
		return "flac"
	default:
		return "mp3"
	}
}

// contentType returns the MIME type for the HTTP stream.
func (s *StreamManager) contentType() string {
	switch s.cfg.StreamFormat {
	case "opus":
		return "audio/ogg"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	default:
		return "audio/mpeg"
	}
}
