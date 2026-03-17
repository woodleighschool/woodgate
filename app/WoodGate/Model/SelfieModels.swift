//
//  SelfieModels.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import CoreMedia
import Foundation
import ImageIO
import Vision

struct CapturedSelfie: Identifiable, Hashable {
  let id = UUID()
  let jpegData: Data
}

struct SelfieAnalysis: Hashable {
  enum Verdict: String {
    case passed
    case unavailable
    case framing
    case alignment
    case visibility
  }

  let verdict: Verdict

  var passed: Bool {
    verdict == .passed
  }

  var message: String {
    switch verdict {
    case .passed:
      "Hold still, then take your selfie."
    case .unavailable:
      "Camera unavailable."
    case .framing:
      "Center your full face in the frame."
    case .alignment:
      "Look straight at the camera."
    case .visibility:
      "Make sure your face is well-lit and unobstructed."
    }
  }
}

enum SelfieAnalyzer {
  nonisolated static func analyze(jpegData: Data) async throws -> SelfieAnalysis {
    try await Task.detached(priority: .userInitiated) {
      try analyzeSync(jpegData: jpegData)
    }.value
  }

  nonisolated static func analyze(
    sampleBuffer: CMSampleBuffer,
    orientation: CGImagePropertyOrientation
  ) throws -> SelfieAnalysis {
    try analyzeSync(sampleBuffer: sampleBuffer, orientation: orientation)
  }

  private nonisolated static func analyzeSync(jpegData: Data) throws -> SelfieAnalysis {
    let source = CGImageSourceCreateWithData(jpegData as CFData, nil)!
    let cgImage = CGImageSourceCreateImageAtIndex(source, 0, nil)!

    return try analyzeSync(cgImage: cgImage, orientation: imageOrientation(from: source))
  }

  private nonisolated static func analyzeSync(
    cgImage: CGImage,
    orientation: CGImagePropertyOrientation
  ) throws -> SelfieAnalysis {
    let handler = VNImageRequestHandler(cgImage: cgImage, orientation: orientation)
    return try analyzeSync(handler: handler)
  }

  private nonisolated static func analyzeSync(
    sampleBuffer: CMSampleBuffer,
    orientation: CGImagePropertyOrientation
  ) throws -> SelfieAnalysis {
    let handler = VNImageRequestHandler(cmSampleBuffer: sampleBuffer, orientation: orientation)
    return try analyzeSync(handler: handler)
  }

  private nonisolated static func analyzeSync(
    handler: VNImageRequestHandler
  ) throws -> SelfieAnalysis {
    let rectanglesRequest = VNDetectFaceRectanglesRequest()
    let landmarksRequest = VNDetectFaceLandmarksRequest()
    let qualityRequest = VNDetectFaceCaptureQualityRequest()

    try handler.perform([rectanglesRequest, landmarksRequest, qualityRequest])

    let faceObservations = rectanglesRequest.results ?? []
    let landmarkObservations = landmarksRequest.results ?? []
    let qualityObservations = qualityRequest.results ?? []

    guard faceObservations.isEmpty == false else {
      return SelfieAnalysis(verdict: .framing)
    }

    guard faceObservations.count == 1 else {
      return SelfieAnalysis(verdict: .framing)
    }

    guard let face = landmarkObservations.first else {
      return SelfieAnalysis(verdict: .framing)
    }

    let area = face.boundingBox.width * face.boundingBox.height
    guard area > 0.08 else {
      return SelfieAnalysis(verdict: .framing)
    }

    if let pitch = face.pitch?.doubleValue, abs(pitch) > 0.5 {
      return SelfieAnalysis(verdict: .alignment)
    }

    if let yaw = face.yaw?.doubleValue, abs(yaw) > 0.5 {
      return SelfieAnalysis(verdict: .alignment)
    }

    if let qualityFace = qualityObservations.first,
       let quality = qualityFace.faceCaptureQuality,
       quality < 0.35
    {
      return SelfieAnalysis(verdict: .visibility)
    }

    guard let landmarks = face.landmarks else {
      return SelfieAnalysis(verdict: .framing)
    }

    guard landmarks.confidence > 0.5 else {
      return SelfieAnalysis(verdict: .visibility)
    }

    let hasEyes =
      hasRegion(landmarks.leftEye, minimumPointCount: 4)
        && hasRegion(landmarks.rightEye, minimumPointCount: 4)
    let hasNose =
      hasRegion(landmarks.nose, minimumPointCount: 3)
        || hasRegion(landmarks.noseCrest, minimumPointCount: 2)
    let hasMouth =
      hasRegion(landmarks.outerLips, minimumPointCount: 6)
        || hasRegion(landmarks.innerLips, minimumPointCount: 4)
    guard hasEyes, hasNose, hasMouth else {
      return SelfieAnalysis(verdict: .framing)
    }

    return SelfieAnalysis(verdict: .passed)
  }

  private nonisolated static func imageOrientation(
    from source: CGImageSource
  ) -> CGImagePropertyOrientation {
    guard
      let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any],
      let orientationValue = properties[kCGImagePropertyOrientation] as? UInt32,
      let orientation = CGImagePropertyOrientation(rawValue: orientationValue)
    else {
      return .up
    }

    return orientation
  }

  private nonisolated static func hasRegion(
    _ region: VNFaceLandmarkRegion2D?,
    minimumPointCount: Int
  ) -> Bool {
    guard let region else { return false }
    return region.pointCount >= minimumPointCount
  }
}
