"use client"

import { useState, useCallback, useEffect, useRef } from "react"
import {
  Room,
  RoomEvent,
  Track,
  type RemoteTrack,
  type LocalTrack,
  type RemoteTrackPublication,
  type RemoteParticipant,
  type LocalTrackPublication,
} from "livekit-client"
import type { CallType } from "../types"

export function useVideoTracks(room: Room | null, callType: CallType) {
  const [remoteVideoTrack, setRemoteVideoTrack] = useState<RemoteTrack | null>(null)
  const [localVideoTrack, setLocalVideoTrack] = useState<LocalTrack | null>(null)
  const remoteElRef = useRef<HTMLVideoElement | null>(null)
  const localElRef = useRef<HTMLVideoElement | null>(null)

  useEffect(() => {
    console.log("[Video] useVideoTracks effect running, room:", !!room, "callType:", callType)
    if (!room || callType !== "video") return

    function onTrackSubscribed(
      track: RemoteTrack,
      _pub: RemoteTrackPublication,
      _participant: RemoteParticipant,
    ) {
      console.log("[Video] TrackSubscribed event:", track.kind, track.sid)
      if (track.kind === Track.Kind.Video) {
        setRemoteVideoTrack(track)
      }
    }

    function onTrackUnsubscribed(track: RemoteTrack) {
      if (track.kind === Track.Kind.Video) {
        track.detach().forEach((el) => el.remove())
        setRemoteVideoTrack(null)
      }
    }

    function onLocalTrackPublished(pub: LocalTrackPublication) {
      console.log("[Video] LocalTrackPublished event:", pub.track?.kind, pub.track?.sid)
      if (pub.track && pub.track.kind === Track.Kind.Video) {
        setLocalVideoTrack(pub.track)
      }
    }

    function onLocalTrackUnpublished(pub: LocalTrackPublication) {
      if (pub.track && pub.track.kind === Track.Kind.Video) {
        pub.track.detach().forEach((el) => el.remove())
        setLocalVideoTrack(null)
      }
    }

    room.on(RoomEvent.TrackSubscribed, onTrackSubscribed)
    room.on(RoomEvent.TrackUnsubscribed, onTrackUnsubscribed)
    room.on(RoomEvent.LocalTrackPublished, onLocalTrackPublished)
    room.on(RoomEvent.LocalTrackUnpublished, onLocalTrackUnpublished)

    // Scan for tracks that were published before this effect ran.
    // This handles the race where connect() resolves and tracks arrive
    // between React's render and useEffect execution.
    for (const participant of room.remoteParticipants.values()) {
      for (const pub of participant.trackPublications.values()) {
        if (pub.track && pub.track.kind === Track.Kind.Video && pub.isSubscribed) {
          console.log("[Video] Scan found remote video track:", pub.track.sid)
          setRemoteVideoTrack(pub.track as RemoteTrack)
        }
      }
    }
    for (const pub of room.localParticipant.trackPublications.values()) {
      if (pub.track && pub.track.kind === Track.Kind.Video) {
        console.log("[Video] Scan found local video track:", pub.track.sid)
        setLocalVideoTrack(pub.track as LocalTrack)
      }
    }

    console.log("[Video] Scan complete - remote participants:", room.remoteParticipants.size, "local pubs:", room.localParticipant.trackPublications.size)

    return () => {
      room.off(RoomEvent.TrackSubscribed, onTrackSubscribed)
      room.off(RoomEvent.TrackUnsubscribed, onTrackUnsubscribed)
      room.off(RoomEvent.LocalTrackPublished, onLocalTrackPublished)
      room.off(RoomEvent.LocalTrackUnpublished, onLocalTrackUnpublished)
    }
  }, [room, callType])

  const attachRemoteVideo = useCallback(
    (el: HTMLVideoElement | null) => {
      if (remoteElRef.current && remoteVideoTrack) {
        remoteVideoTrack.detach(remoteElRef.current)
      }
      remoteElRef.current = el
      if (el && remoteVideoTrack) {
        remoteVideoTrack.attach(el)
      }
    },
    [remoteVideoTrack],
  )

  const attachLocalVideo = useCallback(
    (el: HTMLVideoElement | null) => {
      if (localElRef.current && localVideoTrack) {
        localVideoTrack.detach(localElRef.current)
      }
      localElRef.current = el
      if (el && localVideoTrack) {
        localVideoTrack.attach(el)
      }
    },
    [localVideoTrack],
  )

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (remoteElRef.current && remoteVideoTrack) {
        remoteVideoTrack.detach(remoteElRef.current)
      }
      if (localElRef.current && localVideoTrack) {
        localVideoTrack.detach(localElRef.current)
      }
    }
  }, [remoteVideoTrack, localVideoTrack])

  return {
    remoteVideoTrack,
    localVideoTrack,
    attachRemoteVideo,
    attachLocalVideo,
  }
}
