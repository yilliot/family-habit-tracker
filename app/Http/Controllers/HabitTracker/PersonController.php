<?php

namespace App\Http\Controllers\HabitTracker;

use App\Actions\HabitTracker\CreatePerson;
use App\Actions\HabitTracker\SoftDeletePerson;
use App\Actions\HabitTracker\UpdatePerson;
use App\Http\Controllers\Controller;
use App\Http\Requests\HabitTracker\StorePersonRequest;
use App\Http\Requests\HabitTracker\UpdatePersonRequest;
use App\Models\Person;
use Illuminate\Http\RedirectResponse;

class PersonController extends Controller
{
    public function __construct(
        private CreatePerson $createPerson,
        private UpdatePerson $updatePerson,
        private SoftDeletePerson $softDeletePerson,
    ) {}

    /**
     * Create a new family member.
     */
    public function store(StorePersonRequest $request): RedirectResponse
    {
        $this->createPerson->execute(
            $request->string('name')->toString(),
            $request->integer('display_order') ?: null,
        );

        return to_route('settings.index')->with('success', 'Person added');
    }

    /**
     * Update name and/or display_order on a person.
     */
    public function update(UpdatePersonRequest $request, Person $person): RedirectResponse
    {
        $this->updatePerson->execute($person, $request->validated());

        return to_route('settings.index')->with('success', 'Person updated');
    }

    /**
     * Soft-delete a person and detach them from the active sprint.
     */
    public function destroy(Person $person): RedirectResponse
    {
        $this->softDeletePerson->execute($person);

        return to_route('settings.index')->with('success', 'Person removed');
    }
}
