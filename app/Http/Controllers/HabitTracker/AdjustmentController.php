<?php

namespace App\Http\Controllers\HabitTracker;

use App\Actions\HabitTracker\CreateAdjustment;
use App\Http\Controllers\Controller;
use App\Http\Requests\HabitTracker\StoreAdjustmentRequest;
use App\Models\SprintParticipant;
use Illuminate\Http\RedirectResponse;

class AdjustmentController extends Controller
{
    public function __construct(private CreateAdjustment $createAdjustment) {}

    /**
     * Deduct points from a participant by recording a negative adjustment.
     */
    public function store(StoreAdjustmentRequest $request, SprintParticipant $sprintParticipant): RedirectResponse
    {
        $this->createAdjustment->execute(
            $sprintParticipant,
            -$request->integer('points'),
            $request->string('reason')->toString() ?: null,
        );

        return back()->with('success', 'Points deducted');
    }
}
