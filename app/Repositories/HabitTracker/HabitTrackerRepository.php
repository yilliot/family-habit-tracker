<?php

namespace App\Repositories\HabitTracker;

use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Support\HabitTracker\PointsCalculator;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;

class HabitTrackerRepository
{
    public function __construct(private PointsCalculator $points) {}

    /**
     * The currently-active sprint with eager-loaded participants /
     * habits / ticks / active reward, or null if none is active.
     */
    public function getActiveSprint(): ?Sprint
    {
        return Sprint::query()
            ->active()
            ->with([
                'participants' => fn ($q) => $q->ordered(),
                'participants.person',
                'participants.activeReward',
                'participants.habits' => fn ($q) => $q->ordered(),
                'participants.habits.ticks',
            ])
            ->first();
    }

    /**
     * Returns a render-ready grid for the active sprint, or null when
     * none is active. Shape:
     *
     * @return array{
     *     sprint: array{id:int, start_date:string, end_date:string, status:string},
     *     days: array<int, array{date:string, is_weekend:bool, is_month_start:bool}>,
     *     participants: array<int, array{
     *         id:int,
     *         person:array{id:int, name:string, display_order:int},
     *         display_order:int,
     *         carry_forward_balance:int,
     *         points_balance:int,
     *         active_reward: array{id:int, name:string, cost:int, achieved_at:?string}|null,
     *         habits: array<int, array{id:int, name:string, display_order:int, tick_dates: array<int,string>}>,
     *     }>
     * }|null
     */
    public function getActiveSprintGrid(): ?array
    {
        $sprint = $this->getActiveSprint();

        if ($sprint === null) {
            return null;
        }

        return $this->buildGrid($sprint);
    }

    /**
     * Read-only grid for an archived sprint, same shape as
     * getActiveSprintGrid.
     *
     * @return array<string, mixed>
     */
    public function getArchivedSprintGrid(Sprint $sprint): array
    {
        $sprint->load([
            'participants' => fn ($q) => $q->ordered(),
            'participants.person',
            'participants.activeReward',
            'participants.habits' => fn ($q) => $q->ordered(),
            'participants.habits.ticks',
        ]);

        return $this->buildGrid($sprint);
    }

    /**
     * @return array<string, mixed>
     */
    private function buildGrid(Sprint $sprint): array
    {
        $start = Carbon::parse($sprint->start_date)->startOfDay();
        $end = Carbon::parse($sprint->end_date)->startOfDay();

        $days = [];
        $cursor = $start->copy();
        while ($cursor->lte($end)) {
            $days[] = [
                'date' => $cursor->toDateString(),
                'is_weekend' => $cursor->isWeekend(),
                'is_month_start' => $cursor->day === 1 && ! $cursor->equalTo($start),
            ];
            $cursor->addDay();
        }

        $participants = $sprint->participants->map(function (SprintParticipant $participant): array {
            $habits = $participant->habits->map(function ($habit): array {
                return [
                    'id' => $habit->id,
                    'name' => $habit->name,
                    'display_order' => $habit->display_order,
                    'tick_dates' => $habit->ticks
                        ->map(fn ($t) => Carbon::parse($t->tick_date)->toDateString())
                        ->values()
                        ->all(),
                ];
            })->all();

            $reward = $participant->activeReward;

            return [
                'id' => $participant->id,
                'person' => [
                    'id' => $participant->person->id,
                    'name' => $participant->person->name,
                    'display_order' => $participant->person->display_order,
                ],
                'display_order' => $participant->display_order,
                'carry_forward_balance' => $participant->carry_forward_balance,
                'points_balance' => $this->points->sprintParticipantBalance($participant),
                'active_reward' => $reward === null ? null : [
                    'id' => $reward->id,
                    'name' => $reward->name,
                    'cost' => $reward->cost,
                    'achieved_at' => $reward->achieved_at?->toIso8601String(),
                ],
                'habits' => $habits,
            ];
        })->all();

        return [
            'sprint' => [
                'id' => $sprint->id,
                'start_date' => $start->toDateString(),
                'end_date' => $end->toDateString(),
                'status' => $sprint->status,
            ],
            'days' => $days,
            'participants' => $participants,
        ];
    }

    /**
     * Lifetime points balance for a person.
     */
    public function getPersonBalance(Person $person): int
    {
        return $this->points->lifetimeBalanceForPerson($person);
    }

    /**
     * Sprint-scoped points balance for a participant.
     */
    public function getSprintParticipantBalance(SprintParticipant $participant): int
    {
        return $this->points->sprintParticipantBalance($participant);
    }

    /**
     * Archived sprints, most recent first.
     *
     * @return array<int, array{
     *     id:int, start_date:string, end_date:string, ended_at:?string,
     *     participant_count:int,
     *     achievers: array<int, array{person_id:int, name:string, reward_name:string}>,
     * }>
     */
    public function listArchivedSprints(): array
    {
        $sprints = Sprint::query()
            ->archived()
            ->with([
                'participants.person',
                'participants.activeReward',
            ])
            ->orderByDesc('end_date')
            ->orderByDesc('id')
            ->get();

        return $sprints->map(function (Sprint $sprint): array {
            $sprintStart = $sprint->started_at ?? $sprint->created_at ?? Carbon::parse($sprint->start_date)->startOfDay();
            $sprintEnd = $sprint->ended_at ?? Carbon::parse($sprint->end_date)->endOfDay();

            $achievers = $sprint->participants
                ->filter(fn (SprintParticipant $p) => $p->activeReward !== null
                    && $p->activeReward->achieved_at !== null
                    && $p->activeReward->achieved_at->between($sprintStart, $sprintEnd))
                ->map(fn (SprintParticipant $p) => [
                    'person_id' => $p->person->id,
                    'name' => $p->person->name,
                    'reward_name' => $p->activeReward->name,
                ])
                ->values()
                ->all();

            return [
                'id' => $sprint->id,
                'start_date' => Carbon::parse($sprint->start_date)->toDateString(),
                'end_date' => Carbon::parse($sprint->end_date)->toDateString(),
                'ended_at' => $sprint->ended_at?->toIso8601String(),
                'participant_count' => $sprint->participants->count(),
                'achievers' => $achievers,
            ];
        })->all();
    }

    /**
     * Lifetime totals per non-trashed person.
     *
     * @return array<int, array{
     *     person_id:int,
     *     name:string,
     *     display_order:int,
     *     total_ticks_earned:int,
     *     total_rewards_achieved:int,
     *     sprints_participated_count:int,
     * }>
     */
    public function getLifetimeTotalsPerPerson(): array
    {
        $people = Person::query()->ordered()->get();

        return $people->map(function (Person $person): array {
            $ticks = (int) DB::table('ticks')
                ->join('habits', 'habits.id', '=', 'ticks.habit_id')
                ->join('sprint_participants', 'sprint_participants.id', '=', 'habits.sprint_participant_id')
                ->where('sprint_participants.person_id', $person->id)
                ->count();

            $rewards = (int) Reward::query()
                ->where('person_id', $person->id)
                ->whereNotNull('achieved_at')
                ->count();

            $sprints = (int) SprintParticipant::query()
                ->where('person_id', $person->id)
                ->count();

            return [
                'person_id' => $person->id,
                'name' => $person->name,
                'display_order' => $person->display_order,
                'total_ticks_earned' => $ticks,
                'total_rewards_achieved' => $rewards,
                'sprints_participated_count' => $sprints,
            ];
        })->all();
    }

    /**
     * @return Collection<int, Person>
     */
    public function listPeople(): Collection
    {
        return Person::query()->ordered()->get();
    }

    /**
     * Person rewards, achieved most-recent first followed by the active
     * (un-achieved) reward.
     *
     * @return Collection<int, Reward>
     */
    public function listPersonRewards(Person $person): Collection
    {
        return Reward::query()
            ->where('person_id', $person->id)
            ->orderByRaw('achieved_at IS NULL DESC')
            ->orderByDesc('achieved_at')
            ->orderByDesc('id')
            ->get();
    }
}
