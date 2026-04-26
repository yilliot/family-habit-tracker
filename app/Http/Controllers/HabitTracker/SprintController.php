<?php

namespace App\Http\Controllers\HabitTracker;

use App\Actions\HabitTracker\EndSprint;
use App\Actions\HabitTracker\StartSprint;
use App\Http\Controllers\Controller;
use App\Http\Requests\HabitTracker\StoreSprintRequest;
use App\Models\Sprint;
use Illuminate\Http\RedirectResponse;

class SprintController extends Controller
{
    public function __construct(
        private StartSprint $startSprint,
        private EndSprint $endSprint,
    ) {}

    /**
     * Start a new sprint.
     */
    public function store(StoreSprintRequest $request): RedirectResponse
    {
        $payload = $request->validated();

        $participants = collect($payload['participants'])
            ->map(fn (array $p): array => [
                'person_id' => (int) $p['person_id'],
                'habits' => array_map(
                    fn (array $h): array => [
                        'name' => $h['name'],
                        'display_order' => $h['display_order'] ?? null,
                    ],
                    $p['habits'] ?? [],
                ),
                'reward' => $this->resolveRewardInput($p['reward'] ?? null),
            ])
            ->all();

        $this->startSprint->execute(
            $payload['start_date'],
            $payload['end_date'],
            $participants,
        );

        return to_route('tracker.show')->with('success', 'Sprint started');
    }

    /**
     * Archive the active sprint.
     */
    public function end(Sprint $sprint): RedirectResponse
    {
        $this->endSprint->execute($sprint);

        return to_route('tracker.show')->with('success', 'Sprint archived. You can start a new one.');
    }

    /**
     * Translate the spec's `mode` shape into the StartSprint action's
     * `{name, cost}|null` reward input.
     *
     * - `keep_current` / `none` / null → null (StartSprint links existing
     *    active reward when present, leaves null otherwise).
     * - `new` → { name, cost } so the action creates a new reward.
     *
     * @param  array<string, mixed>|null  $reward
     * @return array{name:string, cost:int}|null
     */
    private function resolveRewardInput(?array $reward): ?array
    {
        if ($reward === null) {
            return null;
        }

        $mode = $reward['mode'] ?? 'keep_current';

        if ($mode === 'new') {
            return [
                'name' => $reward['name'],
                'cost' => (int) $reward['cost'],
            ];
        }

        return null;
    }
}
