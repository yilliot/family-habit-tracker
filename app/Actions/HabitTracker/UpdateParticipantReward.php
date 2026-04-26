<?php

namespace App\Actions\HabitTracker;

use App\Models\Reward;
use App\Models\SprintParticipant;
use Illuminate\Validation\ValidationException;

class UpdateParticipantReward
{
    /**
     * Swap which reward is the "active" one for this participant.
     */
    public function execute(SprintParticipant $participant, ?int $rewardId): SprintParticipant
    {
        $participant->loadMissing('sprint');

        if ($participant->sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Cannot edit participants of an archived sprint.',
            ]);
        }

        if ($rewardId === null) {
            $participant->update(['active_reward_id' => null]);

            return $participant->refresh();
        }

        $reward = Reward::query()->whereKey($rewardId)->firstOrFail();

        if ($reward->person_id !== $participant->person_id) {
            throw ValidationException::withMessages([
                'reward_id' => 'Reward does not belong to this participant.',
            ]);
        }

        if ($reward->isAchieved()) {
            throw ValidationException::withMessages([
                'reward_id' => 'Cannot link an achieved reward.',
            ]);
        }

        $participant->update(['active_reward_id' => $reward->id]);

        return $participant->refresh();
    }
}
