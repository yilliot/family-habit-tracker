import type { ReactNode } from 'react';

type Props = {
    title: string;
    description?: string;
    children?: ReactNode;
};

export default function EmptyState({ title, description, children }: Props) {
    return (
        <div className="ht-empty mx-auto my-12 max-w-xl">
            <h3
                className="ht-mast-title mb-2 text-2xl"
                style={{ fontFamily: 'var(--font-serif-sc)' }}
            >
                {title}
            </h3>
            {description ? <p className="mb-4 text-sm">{description}</p> : null}
            {children ? <div className="mt-4">{children}</div> : null}
        </div>
    );
}
