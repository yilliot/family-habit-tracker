<?php

namespace App\Actions\HabitTracker;

use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Validation\ValidationException;

class DeleteReward
{
    /**
     * Hard-delete an unachieved reward that is not the active reward of
     * any participant in the active sprint.
     */
    public function execute(Reward $reward): void
    {
        if ($reward->isAchieved()) {
            throw ValidationException::withMessages([
                'reward' => 'Cannot delete an achieved reward.',
            ]);
        }

        $activeSprint = Sprint::query()->active()->first();

        if ($activeSprint !== null) {
            $linked = SprintParticipant::query()
                ->where('sprint_id', $activeSprint->id)
                ->where('active_reward_id', $reward->id)
                ->exists();

            if ($linked) {
                throw ValidationException::withMessages([
                    'reward' => 'Cannot delete a reward currently linked to an active sprint participant.',
                ]);
            }
        }

        $reward->delete();
    }
}
