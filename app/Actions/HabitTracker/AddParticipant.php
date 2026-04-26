<?php

namespace App\Actions\HabitTracker;

use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Support\HabitTracker\PointsCalculator;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\ValidationException;

class AddParticipant
{
    public function __construct(private PointsCalculator $points) {}

    /**
     * Add a person mid-sprint to an active Sprint.
     *
     * @param  array<int, array{name: string, display_order?: int}>  $habits
     * @param  array{name: string, cost: int}|null  $reward
     */
    public function execute(Sprint $sprint, int $personId, array $habits = [], ?array $reward = null): SprintParticipant
    {
        if ($sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Cannot add participants to an archived sprint.',
            ]);
        }

        $person = Person::query()->whereKey($personId)->firstOrFail();

        if ($sprint->participants()->where('person_id', $person->id)->exists()) {
            throw ValidationException::withMessages([
                'person_id' => 'This person is already a participant in the sprint.',
            ]);
        }

        return DB::transaction(function () use ($sprint, $person, $habits, $reward): SprintParticipant {
            $resolvedReward = $this->resolveReward($person, $reward);

            $nextOrder = (int) $sprint->participants()->max('display_order') + 1;

            $participant = SprintParticipant::create([
                'sprint_id' => $sprint->id,
                'person_id' => $person->id,
                'carry_forward_balance' => max(0, $this->points->lifetimeBalanceForPerson($person)),
                'active_reward_id' => $resolvedReward?->id,
                'display_order' => $person->display_order ?: $nextOrder,
            ]);

            foreach ($habits as $index => $habit) {
                $participant->habits()->create([
                    'name' => $habit['name'],
                    'display_order' => $habit['display_order'] ?? ($index + 1),
                ]);
            }

            return $participant->load(['person', 'activeReward', 'habits']);
        });
    }

    /**
     * @param  array{name: string, cost: int}|null  $rewardData
     */
    private function resolveReward(Person $person, ?array $rewardData): ?Reward
    {
        $existingActive = $person->rewards()->active()->first();

        if ($rewardData === null) {
            return $existingActive;
        }

        if ($existingActive !== null) {
            return $existingActive;
        }

        return Reward::create([
            'person_id' => $person->id,
            'name' => $rewardData['name'],
            'cost' => $rewardData['cost'],
        ]);
    }
}
