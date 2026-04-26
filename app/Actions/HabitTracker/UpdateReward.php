<?php

namespace App\Actions\HabitTracker;

use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Support\HabitTracker\PointsCalculator;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\ValidationException;

class UpdateReward
{
    public function __construct(private PointsCalculator $points) {}

    /**
     * Update a reward.
     *
     * - Achieved rewards: only `name` is editable. `cost` edits are
     *   rejected.
     * - Active rewards: `name` and `cost` editable. A `cost` change
     *   triggers an achievement check; if any active-sprint participant
     *   linked to this reward now has balance ≥ new cost, the reward is
     *   flipped to achieved.
     *
     * @param  array<string, mixed>  $attributes
     */
    public function execute(Reward $reward, array $attributes): Reward
    {
        $payload = array_intersect_key($attributes, array_flip(['name', 'cost']));

        if ($reward->isAchieved() && array_key_exists('cost', $payload)) {
            throw ValidationException::withMessages([
                'cost' => 'Cannot edit cost on an achieved reward.',
            ]);
        }

        return DB::transaction(function () use ($reward, $payload): Reward {
            $reward->fill($payload);
            $reward->save();

            if ($reward->isAchieved()) {
                return $reward->refresh();
            }

            // Re-check achievement against the active sprint participant
            // who has this as their active reward (if any).
            $activeSprint = Sprint::query()->active()->first();

            if ($activeSprint !== null) {
                $participant = SprintParticipant::query()
                    ->where('sprint_id', $activeSprint->id)
                    ->where('active_reward_id', $reward->id)
                    ->first();

                if ($participant !== null) {
                    $balance = $this->points->sprintParticipantBalance($participant);
                    if ($balance >= $reward->cost) {
                        $reward->update(['achieved_at' => now()]);
                    }
                }
            }

            return $reward->refresh();
        });
    }
}
