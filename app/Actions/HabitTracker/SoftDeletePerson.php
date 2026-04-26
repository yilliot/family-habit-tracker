<?php

namespace App\Actions\HabitTracker;

use App\Models\Person;
use App\Models\Sprint;
use Illuminate\Support\Facades\DB;

class SoftDeletePerson
{
    /**
     * Soft-delete a Person.
     *
     * If the person is part of the active sprint, also detach them by
     * removing their SprintParticipant row (which cascades to their Habits)
     * — but soft-delete the habits first so their tick history is
     * preserved through the soft-deleted habits.
     */
    public function execute(Person $person): Person
    {
        DB::transaction(function () use ($person): void {
            $activeSprint = Sprint::query()->active()->first();

            if ($activeSprint !== null) {
                $participant = $activeSprint->participants()
                    ->where('person_id', $person->id)
                    ->first();

                if ($participant !== null) {
                    // Soft-delete habits (preserves ticks via the soft-deleted habit row).
                    $participant->habits()->each(function ($habit): void {
                        $habit->delete();
                    });

                    // Drop the participant pivot to detach from active sprint.
                    $participant->delete();
                }
            }

            $person->delete();
        });

        return $person;
    }
}
