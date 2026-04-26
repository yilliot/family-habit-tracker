<?php

namespace App\Actions\HabitTracker;

use App\Models\Reward;
use App\Models\SprintParticipant;
use App\Support\HabitTracker\PointsCalculator;

class DetectAchievement
{
    public function __construct(private PointsCalculator $points) {}

    /**
     * Check whether this participant's active reward should flip to achieved
     * based on current balance. Returns the reward if it just became
     * achieved, otherwise null.
     */
    public function execute(SprintParticipant $participant): ?Reward
    {
        $participant->loadMissing('activeReward');

        $reward = $participant->activeReward;

        if ($reward === null || $reward->isAchieved()) {
            return null;
        }

        $balance = $this->points->sprintParticipantBalance($participant);

        if ($balance < $reward->cost) {
            return null;
        }

        $reward->update(['achieved_at' => now()]);

        return $reward->refresh();
    }
}
