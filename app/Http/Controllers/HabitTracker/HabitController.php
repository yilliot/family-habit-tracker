<?php

namespace App\Http\Controllers\HabitTracker;

use App\Actions\HabitTracker\CreateHabit;
use App\Actions\HabitTracker\SoftDeleteHabit;
use App\Actions\HabitTracker\UpdateHabit;
use App\Http\Controllers\Controller;
use App\Http\Requests\HabitTracker\StoreHabitRequest;
use App\Http\Requests\HabitTracker\UpdateHabitRequest;
use App\Models\Habit;
use App\Models\SprintParticipant;
use Illuminate\Http\RedirectResponse;

class HabitController extends Controller
{
    public function __construct(
        private CreateHabit $createHabit,
        private UpdateHabit $updateHabit,
        private SoftDeleteHabit $softDeleteHabit,
    ) {}

    /**
     * Add a habit to a sprint participant.
     */
    public function store(StoreHabitRequest $request, SprintParticipant $sprintParticipant): RedirectResponse
    {
        $this->createHabit->execute(
            $sprintParticipant,
            $request->string('name')->toString(),
            $request->integer('display_order') ?: null,
        );

        return to_route('settings.index')->with('success', 'Habit added');
    }

    /**
     * Update a habit's name and/or display order.
     */
    public function update(UpdateHabitRequest $request, Habit $habit): RedirectResponse
    {
        $this->updateHabit->execute($habit, $request->validated());

        return to_route('settings.index')->with('success', 'Habit updated');
    }

    /**
     * Soft-delete a habit (preserves tick history).
     */
    public function destroy(Habit $habit): RedirectResponse
    {
        $this->softDeleteHabit->execute($habit);

        return to_route('settings.index')->with('success', 'Habit removed');
    }
}
