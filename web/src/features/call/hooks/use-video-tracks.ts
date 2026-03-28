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
    if (!room || callType !== "video") return

    function onTrackSubscribed(
      track: RemoteTrack,
      _pub: RemoteTrackPublication,
      _participant: RemoteParticipant,
    ) {
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
