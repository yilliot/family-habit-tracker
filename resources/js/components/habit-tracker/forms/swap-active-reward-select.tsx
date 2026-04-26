import { router } from '@inertiajs/react';
import sprintParticipants from '@/routes/sprint-participants';
import type { HtReward } from '@/types/habit-tracker';

type Props = {
    sprintParticipantId: number;
    activeRewardId: number | null;
    rewards: HtReward[];
};

export default function SwapActiveRewardSelect({
    sprintParticipantId,
    activeRewardId,
    rewards,
}: Props) {
    const candidates = rewards.filter((r) => r.achieved_at === null);

    if (candidates.length === 0) {
        return (
            <p className="text-xs text-[var(--ink-3)]">
                No unachieved rewards to choose from.
            </p>
        );
    }

    function handleChange(e: React.ChangeEvent<HTMLSelectElement>) {
        const rewardId = Number(e.target.value);

        router.post(
            sprintParticipants.updateReward(sprintParticipantId).url,
            { reward_id: rewardId },
            { preserveScroll: true },
        );
    }

    return (
        <div>
            <label
                className="ht-label"
                htmlFor={`swap-reward-${sprintParticipantId}`}
            >
                Active reward
            </label>
            <select
                id={`swap-reward-${sprintParticipantId}`}
                className="ht-input"
                value={activeRewardId ?? ''}
                onChange={handleChange}
            >
                {activeRewardId === null && (
                    <option value="" disabled>
                        Pick a reward
                    </option>
                )}
                {candidates.map((r) => (
                    <option key={r.id} value={r.id}>
                        {r.name} · {r.cost}pt
                    </option>
                ))}
            </select>
        </div>
    );
}
