<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Inertia\Testing\AssertableInertia;
use Tests\TestCase;

class SettingsControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_index_renders_with_no_active_sprint(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        Person::factory()->create(['name' => '爸爸', 'display_order' => 1]);

        $this->get(route('settings.index'))
            ->assertOk()
            ->assertInertia(fn (AssertableInertia $page) => $page
                ->component('settings/Index')
                ->where('active_sprint', null)
                ->has('people', 1)
                ->has('next_sprint_defaults')
                ->where('today', '2026-04-26'),
            );
    }

    public function test_index_renders_active_sprint_and_pool(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $father = Person::factory()->create(['name' => '爸爸', 'display_order' => 1]);
        $mother = Person::factory()->create(['name' => '妈妈', 'display_order' => 2]);
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-05-16',
        ]);
        SprintParticipant::factory()->for($sprint)->for($father)->create();
        Reward::factory()->for($mother)->create(['name' => 'Walk', 'cost' => 30]);

        $this->get(route('settings.index'))
            ->assertOk()
            ->assertInertia(fn (AssertableInertia $page) => $page
                ->component('settings/Index')
                ->where('active_sprint.sprint.id', $sprint->id)
                ->has('available_for_sprint', 1)
                ->where('available_for_sprint.0.id', $mother->id)
                ->has('rewards_by_person'),
            );
    }
}
