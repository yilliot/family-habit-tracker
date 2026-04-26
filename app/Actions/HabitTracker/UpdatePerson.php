<?php

namespace App\Actions\HabitTracker;

use App\Models\Person;

class UpdatePerson
{
    /**
     * Update name and/or display order on a Person. Other attributes are
     * ignored.
     *
     * @param  array<string, mixed>  $attributes
     */
    public function execute(Person $person, array $attributes): Person
    {
        $person->fill(array_intersect_key($attributes, array_flip(['name', 'display_order'])));
        $person->save();

        return $person->refresh();
    }
}
