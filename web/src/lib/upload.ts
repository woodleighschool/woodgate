export interface UploadProgress {
  loaded: number;
  total: number;
  percent: number;
}

export interface DirectUploadRequest {
  url: string;
  method: "PUT";
  headers: Record<string, string>;
}

export interface DirectUploadTarget {
  upload: {
    strategy: "direct-put";
    url: string;
    method: string;
    headers?: Record<string, string>;
  };
}

export function directUploadRequest(target: DirectUploadTarget): DirectUploadRequest {
  return {
    url: target.upload.url,
    method: "PUT",
    headers: target.upload.headers ?? {},
  };
}

export function uploadDirect({
  request,
  file,
  signal,
  onProgress,
}: {
  request: DirectUploadRequest;
  file: File;
  signal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
}): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(cancelledError());
      return;
    }

    const xhr = new XMLHttpRequest();
    const finish = () => signal?.removeEventListener("abort", abort);
    const abort = () => xhr.abort();

    xhr.upload.addEventListener("progress", (event) => {
      onProgress?.({
        loaded: event.loaded,
        total: file.size,
        percent: file.size > 0 ? Math.round((event.loaded / file.size) * 100) : 0,
      });
    });
    xhr.addEventListener("load", () => {
      finish();
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
        return;
      }
      reject(new Error(`Upload failed with HTTP ${xhr.status}.`));
    });
    xhr.addEventListener("error", () => {
      finish();
      reject(new Error("Upload failed before the storage service accepted the request."));
    });
    xhr.addEventListener("abort", () => {
      finish();
      reject(cancelledError());
    });

    signal?.addEventListener("abort", abort, { once: true });
    xhr.open(request.method, request.url);
    for (const [key, value] of Object.entries(request.headers)) xhr.setRequestHeader(key, value);
    xhr.send(file);
  });
}

function cancelledError() {
  return new Error("Upload cancelled.");
}
