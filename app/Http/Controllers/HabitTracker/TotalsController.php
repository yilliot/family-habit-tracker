<?php

namespace App\Http\Controllers\HabitTracker;

use App\Http\Controllers\Controller;
use App\Repositories\HabitTracker\HabitTrackerRepository;
use Inertia\Inertia;
use Inertia\Response;

class TotalsController extends Controller
{
    public function __construct(private HabitTrackerRepository $repository) {}

    /**
     * Per-person lifetime totals across all sprints.
     */
    public function show(): Response
    {
        return Inertia::render('totals/Show', [
            'people' => $this->repository->getLifetimeTotalsPerPerson(),
        ]);
    }
}
