<?php

namespace App\Support\HabitTracker;

use App\Models\Person;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Repositories\HabitTracker\HabitTrackerRepository;
use Illuminate\Support\Carbon;

/**
 * Builds the auxiliary data the Settings page needs on top of what
 * {@see HabitTrackerRepository} already provides.
 */
class SettingsPresenter
{
    public function __construct(private HabitTrackerRepository $repository) {}

    /**
     * People excluded from the active sprint, in display order. Used by the
     * "Add participant" picker in Settings.
     *
     * @return array<int, array{id:int, name:string, display_order:int}>
     */
    public function availableForSprint(?Sprint $activeSprint): array
    {
        $people = $this->repository->listPeople();

        if ($activeSprint === null) {
            return $people->map(fn (Person $p) => [
                'id' => $p->id,
                'name' => $p->name,
                'display_order' => $p->display_order,
            ])->values()->all();
        }

        $participatingIds = $activeSprint->participants->pluck('person_id')->all();

        return $people
            ->reject(fn (Person $p) => in_array($p->id, $participatingIds, true))
            ->map(fn (Person $p) => [
                'id' => $p->id,
                'name' => $p->name,
                'display_order' => $p->display_order,
            ])
            ->values()
            ->all();
    }

    /**
     * Pre-fills for the "Start new sprint" form when no active sprint exists.
     * Roster + habits are drawn from the most recent archived sprint; rewards
     * default to each person's currently-active reward.
     *
     * @return array{
     *     start_date:string,
     *     end_date:string,
     *     participants: array<int, array{
     *         person_id:int,
     *         person_name:string,
     *         habits: array<int, array{name:string, display_order:int}>,
     *         active_reward: array{id:int, name:string, cost:int}|null,
     *     }>
     * }
     */
    public function nextSprintDefaults(): array
    {
        $today = Carbon::today();

        $latest = Sprint::query()
            ->orderByDesc('end_date')
            ->orderByDesc('id')
            ->with([
                'participants' => fn ($q) => $q->ordered(),
                'participants.person',
                'participants.habits' => fn ($q) => $q->ordered(),
            ])
            ->first();

        $participants = [];

        if ($latest !== null) {
            $participants = $latest->participants
                ->filter(fn (SprintParticipant $p) => $p->person !== null && $p->person->deleted_at === null)
                ->map(function (SprintParticipant $p): array {
                    $person = $p->person;
                    $activeReward = $person->rewards()->active()->first();

                    return [
                        'person_id' => $person->id,
                        'person_name' => $person->name,
                        'habits' => $p->habits->map(fn ($h) => [
                            'name' => $h->name,
                            'display_order' => $h->display_order,
                        ])->values()->all(),
                        'active_reward' => $activeReward === null ? null : [
                            'id' => $activeReward->id,
                            'name' => $activeReward->name,
                            'cost' => $activeReward->cost,
                        ],
                    ];
                })
                ->values()
                ->all();
        }

        return [
            'start_date' => $today->toDateString(),
            'end_date' => $today->copy()->addDays(28)->toDateString(),
            'participants' => $participants,
        ];
    }

    /**
     * Per-person reward catalogue, keyed by person_id, scoped to non-trashed
     * people. Used by the Settings page reward editor.
     *
     * @return array<int, array<int, array{id:int, name:string, cost:int, achieved_at:?string}>>
     */
    public function rewardsByPerson(): array
    {
        $byPerson = [];

        foreach ($this->repository->listPeople() as $person) {
            $byPerson[$person->id] = $this->repository->listPersonRewards($person)
                ->map(fn ($r) => [
                    'id' => $r->id,
                    'name' => $r->name,
                    'cost' => $r->cost,
                    'achieved_at' => $r->achieved_at?->toIso8601String(),
                ])
                ->values()
                ->all();
        }

        return $byPerson;
    }
}
