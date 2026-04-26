<?php

namespace App\Actions\HabitTracker;

use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Support\HabitTracker\PointsCalculator;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\ValidationException;

class StartSprint
{
    public function __construct(private PointsCalculator $points) {}

    /**
     * Start a brand-new sprint with the supplied participant config.
     *
     * @param  array<int, array{
     *     person_id: int,
     *     habits?: array<int, array{name: string, display_order?: int}>,
     *     reward?: array{name: string, cost: int}|null,
     * }>  $participants
     */
    public function execute(string|Carbon $startDate, string|Carbon $endDate, array $participants): Sprint
    {
        $start = Carbon::parse($startDate)->startOfDay();
        $end = Carbon::parse($endDate)->startOfDay();

        if ($end->lt($start)) {
            throw ValidationException::withMessages([
                'end_date' => 'End date must be on or after start date.',
            ]);
        }

        if (Sprint::query()->active()->exists()) {
            throw ValidationException::withMessages([
                'sprint' => 'There is already an active sprint. End it before starting a new one.',
            ]);
        }

        return DB::transaction(function () use ($start, $end, $participants): Sprint {
            $sprint = Sprint::create([
                'start_date' => $start->toDateString(),
                'end_date' => $end->toDateString(),
                'status' => Sprint::STATUS_ACTIVE,
                'started_at' => now(),
            ]);

            foreach ($participants as $index => $participantData) {
                $person = Person::query()->whereKey($participantData['person_id'])->firstOrFail();

                $reward = $this->resolveReward($person, $participantData['reward'] ?? null);

                $participant = SprintParticipant::create([
                    'sprint_id' => $sprint->id,
                    'person_id' => $person->id,
                    'carry_forward_balance' => max(0, $this->points->lifetimeBalanceForPerson($person)),
                    'active_reward_id' => $reward?->id,
                    'display_order' => $person->display_order ?: ($index + 1),
                ]);

                foreach ($participantData['habits'] ?? [] as $habitIndex => $habit) {
                    $participant->habits()->create([
                        'name' => $habit['name'],
                        'display_order' => $habit['display_order'] ?? ($habitIndex + 1),
                    ]);
                }
            }

            return $sprint->load([
                'participants.person',
                'participants.activeReward',
                'participants.habits' => fn ($q) => $q->ordered(),
            ]);
        });
    }

    /**
     * Either link an existing active reward, create a new one, or return
     * null when no reward was supplied.
     *
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
