export default function Footer() {
    const year = new Date().getFullYear();

    return (
        <footer className="ht-footer mt-8 grid grid-cols-1 gap-6 pt-5 md:grid-cols-[1.2fr_1fr_1fr]">
            <div>
                <h4 className="ht-eyebrow mb-2 font-semibold text-[var(--ink)]">
                    A note for the family · 家人的话
                </h4>
                <p>
                    The habits below were chosen by each of us, for ourselves.
                    We cheer each other on, but the ticks belong to the person
                    whose name is on the row. Be kind. Be consistent. A
                    ninety-percent month beats a perfect week.
                </p>
            </div>
            <div>
                <h4 className="ht-eyebrow mb-2 font-semibold text-[var(--ink)]">
                    The shared tablet · 共用平板
                </h4>
                <p>
                    Anyone in the family can walk up and tap their own ticks.
                    Trust is the model. The reward is the goal.
                </p>
            </div>
            <div className="text-right">
                <span
                    className="block font-medium text-[var(--ink)]"
                    style={{
                        fontFamily: 'var(--font-serif-sc)',
                        fontSize: '32px',
                        lineHeight: 1,
                    }}
                >
                    家
                </span>
                <span className="ht-italic mt-1 block text-[14px] text-[var(--ink-2)]">
                    Family Habit Tracker · {year}
                </span>
            </div>
        </footer>
    );
}
