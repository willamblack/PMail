// One scope per view: changing its query invalidates old responses immediately,
// including the debounce interval before the next request is sent.
export function createRequestScope() {
    let controller;
    let disposed = false;

    const invalidate = () => {
        controller?.abort();
        controller = undefined;
    };

    return {
        invalidate,
        start() {
            if (disposed) return null;
            invalidate();
            const request = new AbortController();
            controller = request;
            return {
                signal: request.signal,
                isCurrent: () => !disposed && controller === request,
            };
        },
        dispose() {
            disposed = true;
            invalidate();
        },
    };
}
