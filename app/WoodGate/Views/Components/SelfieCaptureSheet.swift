//
//  SelfieCaptureSheet.swift
//  WoodGate
//
//  Created by Alexander Hyde on 15/3/2026.
//

@preconcurrency import AVFoundation
import Observation
import SwiftUI
import UIKit

struct SelfieCaptureSheet: View {
  // MARK: - Properties

  let onCapture: (CapturedSelfie) -> Void

  @Environment(\.dismiss) private var dismiss

  @State private var model = SelfieCaptureModel()
  @State private var shutterRotation = 0.0

  // MARK: - Body

  var body: some View {
    ZStack {
      CameraPreviewView(session: model.captureSession)
        .ignoresSafeArea()

      VStack(spacing: 0) {
        hintView
        Spacer()
        shutterButton
      }
      .safeAreaPadding(20)
    }
    .task {
      await model.prepare()
    }
    .onChange(of: model.isCapturing) { _, isCapturing in
      if isCapturing {
        shutterRotation = 360
      } else {
        shutterRotation = 0
      }
    }
    .onDisappear {
      model.stop()
    }
  }

  // MARK: - View Builders

  private var hintView: some View {
    Text(model.hint)
      .font(.headline)
      .multilineTextAlignment(.center)
      .padding(.horizontal, 16)
      .padding(.vertical, 14)
      .frame(maxWidth: .infinity)
      .glassEffect(in: .rect(cornerRadius: 18))
  }

  private var shutterButton: some View {
    Button {
      Task {
        guard let selfie = await model.capture() else { return }
        onCapture(selfie)
        dismiss()
      }
    } label: {
      ZStack {
        Circle()
          .fill(.clear)
          .frame(width: 88, height: 88)
          .glassEffect()

        Circle()
          .trim(from: 0.08, to: 0.72)
          .stroke(.white, style: StrokeStyle(lineWidth: 4, lineCap: .round))
          .frame(width: 96, height: 96)
          .rotationEffect(.degrees(shutterRotation))
          .opacity(model.isCapturing ? 1 : 0)
          .animation(
            model.isCapturing
              ? .linear(duration: 0.9).repeatForever(autoreverses: false)
              : .easeOut(duration: 0.2),
            value: shutterRotation
          )

        Circle()
          .fill(.white.opacity(model.canCapture ? 1 : 0.5))
          .frame(width: 72, height: 72)
      }
    }
    .buttonStyle(.plain)
  }
}

@MainActor
@Observable
private final class SelfieCaptureModel {
  // MARK: - Properties

  private(set) var analysis = SelfieAnalysis(
    verdict: .framing
  )
  private(set) var isCapturing = false

  let captureSession: AVCaptureSession

  private let camera = SelfieCameraPipeline()

  private var hasPrepared = false

  // MARK: - Computed Properties

  var hint: String {
    analysis.message
  }

  var canCapture: Bool {
    analysis.passed && !isCapturing
  }

  // MARK: - Initialization

  init() {
    captureSession = camera.captureSession
    camera.onLiveAnalysis = { [weak self] analysis in
      Task { @MainActor in
        self?.apply(analysis)
      }
    }
  }

  // MARK: - Methods

  func prepare() async {
    guard !hasPrepared else {
      camera.start()
      return
    }

    do {
      try await camera.configure()
      hasPrepared = true
      camera.start()
    } catch {
      analysis = SelfieAnalysis(
        verdict: .unavailable
      )
    }
  }

  func stop() {
    camera.stop()
  }

  func capture() async -> CapturedSelfie? {
    guard canCapture else { return nil }
    isCapturing = true

    do {
      let jpegData = try await camera.capturePhoto()
      let analysis = try await SelfieAnalyzer.analyze(jpegData: jpegData)
      guard analysis.passed else {
        self.analysis = analysis
        isCapturing = false
        return nil
      }

      isCapturing = false
      return CapturedSelfie(jpegData: jpegData)
    } catch {
      analysis = SelfieAnalysis(
        verdict: .framing
      )
      isCapturing = false
      return nil
    }
  }

  // MARK: - Private Helpers

  private func apply(_ analysis: SelfieAnalysis) {
    guard !isCapturing else { return }
    self.analysis = analysis
  }
}

private struct CameraPreviewView: UIViewRepresentable {
  // MARK: - Properties

  let session: AVCaptureSession

  // MARK: - UIViewRepresentable

  func makeUIView(context _: Context) -> PreviewView {
    let view = PreviewView()
    view.previewLayer.videoGravity = .resizeAspectFill
    view.previewLayer.session = session
    return view
  }

  func updateUIView(_ uiView: PreviewView, context _: Context) {
    uiView.previewLayer.session = session
  }
}

private final class PreviewView: UIView {
  // MARK: - Overrides

  override class var layerClass: AnyClass {
    AVCaptureVideoPreviewLayer.self
  }

  // MARK: - Computed Properties

  var previewLayer: AVCaptureVideoPreviewLayer {
    layer as! AVCaptureVideoPreviewLayer
  }
}

private final class SelfieCameraPipeline {
  // MARK: - Properties

  nonisolated(unsafe) let captureSession = AVCaptureSession()

  private nonisolated(unsafe) let photoOutput = AVCapturePhotoOutput()
  private nonisolated(unsafe) let videoOutput = AVCaptureVideoDataOutput()

  private let queue = DispatchQueue(label: "au.edu.vic.woodleigh.woodgate.selfie-camera")
  private let previewDelegate = SelfiePreviewFrameDelegate()

  private nonisolated(unsafe) var isConfigured = false
  private nonisolated(unsafe) var photoCaptureDelegate: SelfiePhotoCaptureDelegate?

  nonisolated(unsafe) var onLiveAnalysis: (@Sendable (SelfieAnalysis) -> Void)?

  // MARK: - Methods

  nonisolated func configure() async throws {
    try await withCheckedThrowingContinuation { continuation in
      configure { result in
        continuation.resume(with: result)
      }
    }
  }

  nonisolated func start() {
    queue.async { [captureSession] in
      guard self.isConfigured, !captureSession.isRunning else { return }
      captureSession.startRunning()
    }
  }

  nonisolated func stop() {
    queue.async { [captureSession] in
      guard captureSession.isRunning else { return }
      captureSession.stopRunning()
    }
  }

  nonisolated func capturePhoto() async throws -> Data {
    try await withCheckedThrowingContinuation { continuation in
      capturePhoto { result in
        continuation.resume(with: result)
      }
    }
  }

  // MARK: - Private Helpers

  private nonisolated func configure(completion: @escaping @Sendable (Result<Void, Error>) -> Void) {
    previewDelegate.onAnalysis = { [weak self] analysis in
      self?.onLiveAnalysis?(analysis)
    }

    queue.async { [self, captureSession, photoOutput, videoOutput, previewDelegate] in
      captureSession.beginConfiguration()
      captureSession.sessionPreset = .photo

      do {
        let camera = try SelfieCameraPipeline.frontCamera()
        let input = try AVCaptureDeviceInput(device: camera)

        if captureSession.inputs.isEmpty, captureSession.canAddInput(input) {
          captureSession.addInput(input)
        }
        if !captureSession.outputs.contains(photoOutput), captureSession.canAddOutput(photoOutput) {
          captureSession.addOutput(photoOutput)
        }

        videoOutput.alwaysDiscardsLateVideoFrames = true
        videoOutput.videoSettings = [
          kCVPixelBufferPixelFormatTypeKey as String: kCVPixelFormatType_32BGRA,
        ]
        videoOutput.setSampleBufferDelegate(previewDelegate, queue: queue)

        if !captureSession.outputs.contains(videoOutput), captureSession.canAddOutput(videoOutput) {
          captureSession.addOutput(videoOutput)
        }

        isConfigured = true
        captureSession.commitConfiguration()
        completion(.success(()))
      } catch {
        captureSession.commitConfiguration()
        completion(.failure(error))
      }
    }
  }

  private nonisolated func capturePhoto(
    completion: @escaping @Sendable (Result<Data, Error>) -> Void
  ) {
    queue.async { [self, photoOutput] in
      guard isConfigured else {
        completion(.failure(SelfieCameraError.captureUnavailable))
        return
      }
      guard photoCaptureDelegate == nil else {
        completion(.failure(SelfieCameraError.captureInProgress))
        return
      }

      let delegate = SelfiePhotoCaptureDelegate()
      delegate.onPhotoCapture = { [weak self] result in
        self?.photoCaptureDelegate = nil
        completion(result)
      }
      photoCaptureDelegate = delegate

      let settings = AVCapturePhotoSettings()
      photoOutput.capturePhoto(with: settings, delegate: delegate)
    }
  }

  private nonisolated static func frontCamera() throws -> AVCaptureDevice {
    guard
      let camera = AVCaptureDevice.default(.builtInWideAngleCamera, for: .video, position: .front)
    else {
      throw SelfieCameraError.frontCameraUnavailable
    }

    return camera
  }
}

private final class SelfiePreviewFrameDelegate: NSObject,
  AVCaptureVideoDataOutputSampleBufferDelegate
{
  // MARK: - Properties

  nonisolated(unsafe) var onAnalysis: (@Sendable (SelfieAnalysis) -> Void)?

  private nonisolated(unsafe) var isAnalyzing = false
  private nonisolated(unsafe) var lastAnalysisTime = CFAbsoluteTimeGetCurrent()

  // MARK: - Initialization

  override nonisolated init() {
    super.init()
  }

  // MARK: - AVCaptureVideoDataOutputSampleBufferDelegate

  nonisolated func captureOutput(
    _: AVCaptureOutput,
    didOutput sampleBuffer: CMSampleBuffer,
    from _: AVCaptureConnection
  ) {
    let now = CFAbsoluteTimeGetCurrent()
    guard !isAnalyzing, now - lastAnalysisTime >= 0.25 else { return }

    isAnalyzing = true
    lastAnalysisTime = now
    defer { isAnalyzing = false }

    guard
      let analysis = try? SelfieAnalyzer.analyze(
        sampleBuffer: sampleBuffer,
        orientation: .leftMirrored
      )
    else {
      return
    }

    onAnalysis?(analysis)
  }
}

private final class SelfiePhotoCaptureDelegate: NSObject, AVCapturePhotoCaptureDelegate {
  // MARK: - Properties

  nonisolated(unsafe) var onPhotoCapture: (@Sendable (Result<Data, Error>) -> Void)?

  // MARK: - Initialization

  override nonisolated init() {
    super.init()
  }

  // MARK: - AVCapturePhotoCaptureDelegate

  nonisolated func photoOutput(
    _: AVCapturePhotoOutput,
    didFinishProcessingPhoto photo: AVCapturePhoto,
    error: Error?
  ) {
    if let error {
      onPhotoCapture?(.failure(error))
      return
    }

    let jpegData = photo.fileDataRepresentation()!
    onPhotoCapture?(.success(jpegData))
  }
}

private enum SelfieCameraError: Error {
  case frontCameraUnavailable
  case captureUnavailable
  case captureInProgress
}
