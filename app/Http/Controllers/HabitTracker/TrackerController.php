<?php

namespace App\Http\Controllers\HabitTracker;

use App\Http\Controllers\Controller;
use App\Repositories\HabitTracker\HabitTrackerRepository;
use Illuminate\Support\Carbon;
use Inertia\Inertia;
use Inertia\Response;

class TrackerController extends Controller
{
    public function __construct(private HabitTrackerRepository $repository) {}

    /**
     * Render the active sprint grid (or an empty state if none).
     */
    public function show(): Response
    {
        $grid = $this->repository->getActiveSprintGrid();

        return Inertia::render('tracker/Show', [
            'sprint' => $grid['sprint'] ?? null,
            'days' => $grid['days'] ?? [],
            'participants' => $grid['participants'] ?? [],
            'today' => Carbon::today()->toDateString(),
        ]);
    }
}
