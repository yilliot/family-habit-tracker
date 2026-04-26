import { usePage } from '@inertiajs/react';
import { useEffect, useRef } from 'react';
import { toast } from 'sonner';
import type { HtFlash } from '@/types/habit-tracker';

/**
 * Surfaces shared `flash.success` / `flash.error` props as toasts.
 *
 * Called once from the habit-tracker layout. Tracks the last seen message
 * via a ref so a successful redirect-back-with-flash fires exactly one
 * toast even if React re-renders before the next props change.
 */
export function useFlashFeedback(): void {
    const page = usePage<{ flash?: HtFlash }>();
    const flash = page.props.flash;
    const lastSuccess = useRef<string | null>(null);
    const lastError = useRef<string | null>(null);

    useEffect(() => {
        const successMsg = flash?.success ?? null;

        if (successMsg && successMsg !== lastSuccess.current) {
            lastSuccess.current = successMsg;
            toast.success(successMsg);
        } else if (!successMsg) {
            lastSuccess.current = null;
        }
    }, [flash?.success]);

    useEffect(() => {
        const errorMsg = flash?.error ?? null;

        if (errorMsg && errorMsg !== lastError.current) {
            lastError.current = errorMsg;
            toast.error(errorMsg);
        } else if (!errorMsg) {
            lastError.current = null;
        }
    }, [flash?.error]);
}
