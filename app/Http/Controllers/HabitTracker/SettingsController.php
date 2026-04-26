<?php

namespace App\Http\Controllers\HabitTracker;

use App\Http\Controllers\Controller;
use App\Models\Person;
use App\Repositories\HabitTracker\HabitTrackerRepository;
use App\Support\HabitTracker\SettingsPresenter;
use Illuminate\Support\Carbon;
use Inertia\Inertia;
use Inertia\Response;

class SettingsController extends Controller
{
    public function __construct(
        private HabitTrackerRepository $repository,
        private SettingsPresenter $presenter,
    ) {}

    /**
     * Render the settings page (people, active sprint config, rewards).
     */
    public function index(): Response
    {
        $activeSprint = $this->repository->getActiveSprint();
        $grid = $activeSprint !== null ? $this->repository->getArchivedSprintGrid($activeSprint) : null;

        $people = Person::query()->ordered()->withTrashed()->get()->map(fn (Person $p) => [
            'id' => $p->id,
            'name' => $p->name,
            'display_order' => $p->display_order,
            'deleted_at' => $p->deleted_at?->toIso8601String(),
        ])->values()->all();

        return Inertia::render('settings/Index', [
            'people' => $people,
            'active_sprint' => $grid,
            'available_for_sprint' => $this->presenter->availableForSprint($activeSprint),
            'rewards_by_person' => $this->presenter->rewardsByPerson(),
            'next_sprint_defaults' => $this->presenter->nextSprintDefaults(),
            'today' => Carbon::today()->toDateString(),
        ]);
    }
}
