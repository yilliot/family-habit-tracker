<?php

namespace App\Actions\HabitTracker;

use App\Models\Person;

class CreatePerson
{
    /**
     * Create a new globally-scoped Person.
     */
    public function execute(string $name, ?int $displayOrder = null): Person
    {
        return Person::create([
            'name' => $name,
            'display_order' => $displayOrder ?? $this->nextDisplayOrder(),
        ]);
    }

    private function nextDisplayOrder(): int
    {
        return (int) Person::query()->max('display_order') + 1;
    }
}
