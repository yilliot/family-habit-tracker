<?php

namespace App\Support\HabitTracker;

use App\Models\Person;
use App\Models\Reward;
use App\Models\SprintParticipant;
use App\Models\Tick;

/**
 * Computes derived points balances on demand. Never persists totals.
 *
 * Lifetime balance for a person = (total ticks ever earned across all habits,
 * including soft-deleted habits) − (sum of cost of all achieved rewards).
 *
 * Sprint-scoped balance for a participant = carry_forward_balance + (ticks
 * inside this participant's habits) − (cost of rewards achieved AT-OR-AFTER
 * this sprint's start).
 */
class PointsCalculator
{
    /**
     * Lifetime balance for a person, ignoring sprint scope. Used by
     * carry-forward seeding when no participant context is available.
     */
    public function lifetimeBalanceForPerson(Person $person): int
    {
        $earned = Tick::query()
            ->whereIn(
                'habit_id',
                fn ($q) => $q->from('habits')
                    ->select('habits.id')
                    ->join('sprint_participants', 'sprint_participants.id', '=', 'habits.sprint_participant_id')
                    ->where('sprint_participants.person_id', $person->id)
            )
            ->count();

        $spent = (int) Reward::query()
            ->where('person_id', $person->id)
            ->whereNotNull('achieved_at')
            ->sum('cost');

        return (int) $earned - $spent;
    }

    /**
     * Sprint-scoped balance for a participant.
     */
    public function sprintParticipantBalance(SprintParticipant $participant): int
    {
        $participant->loadMissing('sprint');

        $earnedInSprint = Tick::query()
            ->whereIn(
                'habit_id',
                fn ($q) => $q->from('habits')
                    ->select('id')
                    ->where('sprint_participant_id', $participant->id)
            )
            ->count();

        $sprintStartedAt = $participant->sprint->started_at
            ?? $participant->sprint->created_at
            ?? $participant->sprint->start_date->startOfDay();

        $spentDuringOrAfterSprintStart = (int) Reward::query()
            ->where('person_id', $participant->person_id)
            ->whereNotNull('achieved_at')
            ->where('achieved_at', '>=', $sprintStartedAt)
            ->sum('cost');

        return (int) $participant->carry_forward_balance + (int) $earnedInSprint - $spentDuringOrAfterSprintStart;
    }
}
