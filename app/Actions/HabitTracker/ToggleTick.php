<?php

namespace App\Actions\HabitTracker;

use App\Models\Habit;
use App\Models\Reward;
use App\Models\Tick;
use App\Support\HabitTracker\PointsCalculator;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\ValidationException;

class ToggleTick
{
    public function __construct(
        private PointsCalculator $points,
        private DetectAchievement $detectAchievement,
    ) {}

    /**
     * Toggle a tick on/off for a (habit, date) pair. Returns the new state
     * along with the participant's post-mutation balance and any reward
     * that just flipped to achieved.
     *
     * @return array{tick: Tick|null, balance_after: int, just_achieved_reward: Reward|null}
     */
    public function execute(Habit $habit, string|Carbon $tickDate): array
    {
        $date = Carbon::parse($tickDate)->startOfDay();
        $today = now()->startOfDay();

        $habit->loadMissing('sprintParticipant.sprint');

        $sprint = $habit->sprintParticipant->sprint;

        if ($sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Cannot tick on an archived sprint.',
            ]);
        }

        if ($date->gt($today)) {
            throw ValidationException::withMessages([
                'tick_date' => 'Cannot tick a future day.',
            ]);
        }

        $sprintStart = Carbon::parse($sprint->start_date)->startOfDay();
        $sprintEnd = Carbon::parse($sprint->end_date)->startOfDay();

        if ($date->lt($sprintStart) || $date->gt($sprintEnd)) {
            throw ValidationException::withMessages([
                'tick_date' => 'Tick date is outside the sprint range.',
            ]);
        }

        return DB::transaction(function () use ($habit, $date) {
            $existing = $habit->ticks()->whereDate('tick_date', $date->toDateString())->first();

            if ($existing !== null) {
                $existing->delete();
                $tick = null;
            } else {
                $tick = $habit->ticks()->create([
                    'tick_date' => $date->toDateString(),
                    'created_at' => now(),
                ]);
            }

            $participant = $habit->sprintParticipant;
            $justAchieved = $tick !== null
                ? $this->detectAchievement->execute($participant)
                : null;

            $balance = $this->points->sprintParticipantBalance($participant);

            return [
                'tick' => $tick,
                'balance_after' => $balance,
                'just_achieved_reward' => $justAchieved,
            ];
        });
    }
}
