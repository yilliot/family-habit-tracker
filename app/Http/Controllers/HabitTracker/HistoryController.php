<?php

namespace App\Http\Controllers\HabitTracker;

use App\Http\Controllers\Controller;
use App\Models\Sprint;
use App\Repositories\HabitTracker\HabitTrackerRepository;
use Illuminate\Support\Carbon;
use Inertia\Inertia;
use Inertia\Response;
use Symfony\Component\HttpKernel\Exception\NotFoundHttpException;

class HistoryController extends Controller
{
    public function __construct(private HabitTrackerRepository $repository) {}

    /**
     * Index of archived sprints.
     */
    public function index(): Response
    {
        return Inertia::render('history/Index', [
            'sprints' => $this->repository->listArchivedSprints(),
        ]);
    }

    /**
     * Read-only grid view of an archived sprint.
     */
    public function show(Sprint $sprint): Response
    {
        if (! $sprint->isArchived()) {
            throw new NotFoundHttpException;
        }

        $grid = $this->repository->getArchivedSprintGrid($sprint);

        return Inertia::render('history/Show', [
            'sprint' => $grid['sprint'],
            'days' => $grid['days'],
            'participants' => $grid['participants'],
            'today' => Carbon::today()->toDateString(),
        ]);
    }
}
