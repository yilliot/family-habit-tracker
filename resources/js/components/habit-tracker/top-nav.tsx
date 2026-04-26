import { Link, usePage } from '@inertiajs/react';
import history from '@/routes/history';
import settings from '@/routes/settings';
import totals from '@/routes/totals';
import tracker from '@/routes/tracker';

const NAV_LINKS: Array<{
    label: string;
    href: () => { url: string };
    matchUrl: string;
}> = [
    { label: 'Tracker', href: tracker.show, matchUrl: tracker.show.url() },
    { label: 'History', href: history.index, matchUrl: history.index.url() },
    { label: 'Totals', href: totals.show, matchUrl: totals.show.url() },
    { label: 'Settings', href: settings.index, matchUrl: settings.index.url() },
];

export default function TopNav() {
    const { url } = usePage();
    // url has the path-with-query — only match the path portion.
    const path = url.split('?')[0];

    return (
        <nav
            aria-label="Primary"
            className="flex flex-wrap items-center gap-1 md:gap-2"
        >
            {NAV_LINKS.map((item) => {
                const active =
                    item.matchUrl === '/'
                        ? path === '/'
                        : path === item.matchUrl ||
                          path.startsWith(item.matchUrl + '/');

                return (
                    <Link
                        key={item.label}
                        href={item.href().url}
                        className="ht-nav-link"
                        data-active={active}
                        prefetch
                    >
                        {item.label}
                    </Link>
                );
            })}
        </nav>
    );
}
