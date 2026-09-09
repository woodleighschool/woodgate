import { type MutationKey, useMutation } from "@tanstack/react-query";
import { type UploadProgress, type UploadRequest, upload } from "@woodleighschool/bloby-client";
import { useRef, useState } from "react";

import { toast } from "@components/ui/toast";

interface UploadOptions<TIntent, TResult, TVars extends { file: File }> {
  mutationKey: MutationKey;
  createIntent: (vars: TVars, signal: AbortSignal) => Promise<TIntent>;
  uploadRequest: (intent: TIntent, vars: TVars) => UploadRequest;
  completeUpload: (intent: TIntent, vars: TVars, signal: AbortSignal) => Promise<TResult>;
  onSuccess?: (result: TResult, vars: TVars) => void | Promise<void>;
  loadingText: string;
  successText: string;
}

export function useUpload<TIntent, TResult, TVars extends { file: File }>({
  mutationKey,
  createIntent,
  uploadRequest,
  completeUpload,
  onSuccess,
  loadingText,
  successText,
}: UploadOptions<TIntent, TResult, TVars>) {
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const uploadAbort = useRef<AbortController | null>(null);

  const mutation = useMutation<TResult, Error, TVars>({
    mutationKey,
    onSuccess,
    mutationFn: async (vars) => {
      const abortController = new AbortController();
      uploadAbort.current = abortController;
      setProgress({ loaded: 0, total: vars.file.size, percent: 0 });
      const toastID = toast.add({
        title: loadingText,
        description: "Preparing upload",
        type: "loading",
        timeout: 0,
      });

      try {
        const intent = await createIntent(vars, abortController.signal);
        abortController.signal.throwIfAborted();
        await upload({
          ...uploadRequest(intent, vars),
          blob: vars.file,
          signal: abortController.signal,
          onProgress: (next) => {
            setProgress(next);
            toast.update(toastID, {
              title: loadingText,
              description: next.percent > 0 ? `${next.percent}%` : "Uploading",
              type: "loading",
              timeout: 0,
            });
          },
        });
        setProgress({ loaded: vars.file.size, total: vars.file.size, percent: 100 });
        toast.update(toastID, {
          title: loadingText,
          description: "Finalizing",
          type: "loading",
          timeout: 0,
        });
        abortController.signal.throwIfAborted();
        const result = await completeUpload(intent, vars, abortController.signal);
        toast.update(toastID, {
          title: successText,
          description: undefined,
          type: "success",
          timeout: 5000,
        });
        return result;
      } catch (error) {
        toast.update(toastID, {
          title: `${loadingText} Failed`,
          description: error instanceof Error ? error.message : "Unknown upload error.",
          type: "error",
          timeout: 5000,
        });
        throw error;
      } finally {
        if (uploadAbort.current === abortController) uploadAbort.current = null;
        setProgress(null);
      }
    },
  });

  return {
    progress,
    mutation,
    upload: mutation.mutateAsync,
    cancel: () => uploadAbort.current?.abort(),
    isUploading: mutation.isPending,
  };
}
